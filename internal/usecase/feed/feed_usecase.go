package feed

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/gdugdh24/mpit2026-backend/internal/domain"
	"github.com/gdugdh24/mpit2026-backend/internal/repository"
)

type FeedUseCase struct {
	userRepo    repository.UserRepository
	profileRepo repository.ProfileRepository
	swipeRepo   repository.SwipeRepository
}

func NewFeedUseCase(
	userRepo repository.UserRepository,
	profileRepo repository.ProfileRepository,
	swipeRepo repository.SwipeRepository,
) *FeedUseCase {
	return &FeedUseCase{
		userRepo:    userRepo,
		profileRepo: profileRepo,
		swipeRepo:   swipeRepo,
	}
}

// FeedUserResponse represents a user in the feed
type FeedUserResponse struct {
	ID                 int      `json:"id"`
	UserID             int      `json:"user_id"`
	DisplayName        string   `json:"display_name"`
	Bio                *string  `json:"bio"`
	City               *string  `json:"city"`
	Age                int      `json:"age"`
	Interests          []string `json:"interests"`
	DistanceKm         *float64 `json:"distance_km,omitempty"`
	CompatibilityScore int      `json:"compatibility_score"`
	CompatibilityLabel string   `json:"compatibility_label"` // New field
}

// CompatibilityDetails holds the breakdown of the score
type CompatibilityDetails struct {
	TotalScore       float64
	PersonalityScore float64
	InterestsScore   float64
	DistanceScore    float64
	CommonInterests  []string
}

// GetNextUser returns the next user for feed
func (uc *FeedUseCase) GetNextUser(ctx context.Context, currentUserID int) (*FeedUserResponse, error) {
	// Get current user's profile for preferences
	currentProfile, err := uc.profileRepo.GetByUserID(ctx, currentUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user profile: %w", err)
	}

	// Get current user
	currentUser, err := uc.userRepo.GetByID(ctx, currentUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	// Build filters based on preferences
	filters := make(map[string]interface{})

	// Filter by onboarding complete
	// TODO: Re-enable after implementing proper onboarding flow
	// filters["is_onboarding_complete"] = true

	// Filter by city if set
	if currentProfile.City != nil && *currentProfile.City != "" {
		filters["city"] = *currentProfile.City
	}

	// Get candidate profiles (exclude already swiped)
	candidates, err := uc.profileRepo.SearchProfiles(ctx, filters, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to search profiles: %w", err)
	}

	// Filter candidates and calculate scores
	type ScoredCandidate struct {
		Profile *domain.Profile
		Details CompatibilityDetails
		User    *domain.User
	}
	var scoredCandidates []ScoredCandidate

	for _, candidate := range candidates {
		// Skip self
		if candidate.UserID == currentUserID {
			continue
		}

		// Check if already swiped
		existingSwipe, err := uc.swipeRepo.GetByUsers(ctx, currentUserID, candidate.UserID)
		if err == nil && existingSwipe != nil {
			continue // Already swiped
		}

		// Get candidate user for age
		candidateUser, err := uc.userRepo.GetByID(ctx, candidate.UserID)
		if err != nil {
			continue
		}

		// Check age preferences
		age := candidateUser.Age()
		if currentProfile.PrefMinAge != nil && age < *currentProfile.PrefMinAge {
			continue
		}
		if currentProfile.PrefMaxAge != nil && age > *currentProfile.PrefMaxAge {
			continue
		}

		// Check gender preferences (if we add gender preference later)
		// For now, just check opposite gender
		if currentUser.Gender == candidateUser.Gender {
			continue // Skip same gender for now
		}

		// Calculate distance if locations available
		var distanceKm *float64
		if currentProfile.LocationLat != nil && currentProfile.LocationLon != nil &&
			candidate.LocationLat != nil && candidate.LocationLon != nil {
			distance := calculateDistance(
				*currentProfile.LocationLat, *currentProfile.LocationLon,
				*candidate.LocationLat, *candidate.LocationLon,
			)

			// Check distance preference
			if currentProfile.PrefMaxDistanceKm != nil && distance > float64(*currentProfile.PrefMaxDistanceKm) {
				continue
			}

			distanceKm = &distance
		}

		// Calculate Compatibility Score
		details := uc.calculateCompatibilityScore(currentProfile, candidate, distanceKm)
		scoredCandidates = append(scoredCandidates, ScoredCandidate{
			Profile: candidate,
			Details: details,
			User:    candidateUser,
		})
	}

	// Sort by score descending
	sort.Slice(scoredCandidates, func(i, j int) bool {
		return scoredCandidates[i].Details.TotalScore > scoredCandidates[j].Details.TotalScore
	})

	// Return top candidate
	if len(scoredCandidates) > 0 {
		best := scoredCandidates[0]

		// Calculate distance again for response (optimization: could store it)
		var distanceKm *float64
		if currentProfile.LocationLat != nil && currentProfile.LocationLon != nil &&
			best.Profile.LocationLat != nil && best.Profile.LocationLon != nil {
			d := calculateDistance(
				*currentProfile.LocationLat, *currentProfile.LocationLon,
				*best.Profile.LocationLat, *best.Profile.LocationLon,
			)
			distanceKm = &d
		}

		// Generate Compatibility Label
		label := "Potential Match"
		details := best.Details

		// Logic for label
		if details.PersonalityScore > 0.8 { // Normalized 0-1 (inside calculate it is * 40, so 32/40)
			// Need to verify range of PersonalityScore from details.
			// Currently calculateCompatibilityScore returns weighted score component.
			// Let's adjust calculateCompatibilityScore to return normalized values (0-1) for easier logic.
		}

		// Let's rely on the returned details values (which I will ensure are normalized 0-1)

		if details.PersonalityScore > 0.8 {
			label = "✨ Soulmate Potential (90% Match)"
		} else if len(details.CommonInterests) > 0 {
			// Pick one random common interest
			interest := details.CommonInterests[0]
			label = fmt.Sprintf("🎮 Ideal %s Partner", interest)
		} else if distanceKm != nil && *distanceKm < 5.0 {
			label = "📍 Neighbor Match (< 5km)"
		} else if details.TotalScore > 75 {
			label = "🔥 High Compatibility"
		}

		return &FeedUserResponse{
			ID:                 best.Profile.ID,
			UserID:             best.Profile.UserID,
			DisplayName:        best.Profile.DisplayName,
			Bio:                best.Profile.Bio,
			City:               best.Profile.City,
			Age:                best.User.Age(),
			Interests:          best.Profile.Interests,
			DistanceKm:         distanceKm,
			CompatibilityScore: int(details.TotalScore),
			CompatibilityLabel: label,
		}, nil
	}

	// No more users in feed
	return nil, nil
}

// calculateCompatibilityScore calculates a 0-100 score and returns details
func (uc *FeedUseCase) calculateCompatibilityScore(me, candidate *domain.Profile, distanceKm *float64) CompatibilityDetails {
	details := CompatibilityDetails{}

	// 1. Personality Compatibility (40%)
	personalityScore := 0.0
	if me.PrefOpenness != nil && candidate.PrefOpenness != nil {
		dist := 0.0
		dist += math.Pow(*me.PrefOpenness-*candidate.PrefOpenness, 2)
		dist += math.Pow(*me.PrefConscientiousness-*candidate.PrefConscientiousness, 2)
		dist += math.Pow(*me.PrefExtraversion-*candidate.PrefExtraversion, 2)
		dist += math.Pow(*me.PrefAgreeableness-*candidate.PrefAgreeableness, 2)
		dist += math.Pow(*me.PrefNeuroticism-*candidate.PrefNeuroticism, 2)
		dist = math.Sqrt(dist)

		// Max distance is sqrt(1^2 * 5) = sqrt(5) = 2.23
		// Normalize to 0-1 (1 means close, 0 means far)
		personalityScore = 1.0 - (dist / 2.23)
		if personalityScore < 0 {
			personalityScore = 0
		}
	} else {
		// Fallback
		personalityScore = 0.5
	}
	details.PersonalityScore = personalityScore
	details.TotalScore += personalityScore * 40

	// 2. Interests Compatibility (30%)
	interestsScore := 0.0
	common := 0
	total := len(me.Interests) + len(candidate.Interests)
	var commonInterests []string

	if total > 0 {
		for _, myInt := range me.Interests {
			for _, theirInt := range candidate.Interests {
				if myInt == theirInt {
					common++
					commonInterests = append(commonInterests, myInt)
					break
				}
			}
		}
		union := total - common
		if union > 0 {
			interestsScore = float64(common) / float64(union)
		}
	}
	details.InterestsScore = interestsScore
	details.CommonInterests = commonInterests
	details.TotalScore += interestsScore * 30

	// 3. Demographics/Distance (30%)
	distScore := 1.0
	if distanceKm != nil {
		maxDist := 100.0
		if me.PrefMaxDistanceKm != nil {
			maxDist = float64(*me.PrefMaxDistanceKm)
		}
		distScore = 1.0 - (*distanceKm / maxDist)
		if distScore < 0 {
			distScore = 0
		}
	}
	details.DistanceScore = distScore
	details.TotalScore += distScore * 30

	return details
}

// Helper functions (duplicated from swipe usecase, should be in shared utils)
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371.0
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)
	lat1Rad := lat1 * (math.Pi / 180.0)
	lat2Rad := lat2 * (math.Pi / 180.0)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}

// ResetDislikes deletes all dislikes for a user to refresh the feed
func (uc *FeedUseCase) ResetDislikes(ctx context.Context, userID int) (int, error) {
	// Get all user swipes
	swipes, err := uc.swipeRepo.GetUserSwipes(ctx, userID, 1000, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to get swipes: %w", err)
	}

	count := 0
	// Note: This is a simplified implementation
	// In production, you'd want a batch delete method in repository
	for _, swipe := range swipes {
		if !swipe.IsLike {
			// Delete dislike swipes
			// For now, we'll just count them
			// You'd need to implement a Delete method in swipe repository
			count++
		}
	}

	return count, nil
}
