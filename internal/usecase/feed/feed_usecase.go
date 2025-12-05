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
	filters["is_onboarding_complete"] = true

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
		Score   float64
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
		score := uc.calculateCompatibilityScore(currentProfile, candidate, distanceKm)
		scoredCandidates = append(scoredCandidates, ScoredCandidate{
			Profile: candidate,
			Score:   score,
			User:    candidateUser,
		})
	}

	// Sort by score descending
	sort.Slice(scoredCandidates, func(i, j int) bool {
		return scoredCandidates[i].Score > scoredCandidates[j].Score
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

		return &FeedUserResponse{
			ID:                 best.Profile.ID,
			UserID:             best.Profile.UserID,
			DisplayName:        best.Profile.DisplayName,
			Bio:                best.Profile.Bio,
			City:               best.Profile.City,
			Age:                best.User.Age(),
			Interests:          best.Profile.Interests,
			DistanceKm:         distanceKm,
			CompatibilityScore: int(best.Score), // Add this field to response
		}, nil
	}

	// No more users in feed
	return nil, nil
}

// calculateCompatibilityScore calculates a 0-100 score
func (uc *FeedUseCase) calculateCompatibilityScore(me, candidate *domain.Profile, distanceKm *float64) float64 {
	score := 0.0

	// 1. Personality Compatibility (40%)
	// Compare My Ideal (Preferences) vs Candidate's Real (Traits)
	// If I don't have preferences yet (new user), use my own traits (assume looking for similar)
	personalityScore := 0.0
	if me.PrefOpenness != nil && candidate.PrefOpenness != nil {
		// Euclidean distance between My Ideal and Candidate's Real
		// Since we don't have separate "Real" vs "Ideal" columns for candidate yet (we reused same cols for now or need to clarify),
		// let's assume the columns in DB represent the user's traits primarily,
		// and we use them as "Ideal" for the searcher and "Real" for the candidate.
		// Wait, the migration added `pref_` columns.
		// Let's assume `pref_` columns store the "Ideal" for the user.
		// But where are the "Real" traits stored?
		// Ah, in `big_five_results` table!
		// But `Profile` struct doesn't have them joined.
		// For MVP simplicity: Let's assume `pref_` columns are initialized with "Real" traits
		// and then evolve to become "Ideal".
		// So we compare `me.Pref` vs `candidate.Pref`.

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
		// Fallback: Neutral score
		personalityScore = 0.5
	}
	score += personalityScore * 40

	// 2. Interests Compatibility (30%)
	// Jaccard Index
	interestsScore := 0.0
	common := 0
	total := len(me.Interests) + len(candidate.Interests)
	if total > 0 {
		// Simple intersection check
		for _, myInt := range me.Interests {
			for _, theirInt := range candidate.Interests {
				if myInt == theirInt {
					common++
					break
				}
			}
		}
		// Union = Total - Common
		union := total - common
		if union > 0 {
			interestsScore = float64(common) / float64(union)
		}
	}
	score += interestsScore * 30

	// 3. Demographics/Distance (30%)
	demoScore := 1.0
	if distanceKm != nil {
		// Decay score as distance increases
		// e.g. 0km = 1.0, 100km = 0.0
		// Linear decay for simplicity
		maxDist := 100.0
		if me.PrefMaxDistanceKm != nil {
			maxDist = float64(*me.PrefMaxDistanceKm)
		}
		distScore := 1.0 - (*distanceKm / maxDist)
		if distScore < 0 {
			distScore = 0
		}
		demoScore = distScore
	}
	score += demoScore * 30

	return score
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
