package profile

import (
	"context"
	"fmt"
	"math"

	"github.com/gdugdh24/mpit2026-backend/internal/domain"
	"github.com/gdugdh24/mpit2026-backend/internal/infrastructure/gemini"
	"github.com/gdugdh24/mpit2026-backend/internal/repository"
)

type ProfileUseCase struct {
	profileRepo  repository.ProfileRepository
	userRepo     repository.UserRepository
	geminiClient *gemini.GeminiClient
}

func NewProfileUseCase(
	profileRepo repository.ProfileRepository,
	userRepo repository.UserRepository,
	geminiClient *gemini.GeminiClient,
) *ProfileUseCase {
	return &ProfileUseCase{
		profileRepo:  profileRepo,
		userRepo:     userRepo,
		geminiClient: geminiClient,
	}
}

// ... existing structs ...

// GenerateBioRequest represents request to generate bio
type GenerateBioRequest struct {
	DisplayName string   `json:"display_name" binding:"required"`
	Interests   []string `json:"interests" binding:"required"`
	City        string   `json:"city" binding:"required"`
}

// GenerateBio generates creative bios
func (uc *ProfileUseCase) GenerateBio(ctx context.Context, req *GenerateBioRequest) (map[string]string, error) {
	if uc.geminiClient == nil {
		return nil, fmt.Errorf("gemini client is not initialized")
	}
	return uc.geminiClient.GenerateBio(ctx, req.DisplayName, req.Interests, req.City)
}

// ... existing methods ...

// calculateDistance uses stdlib math
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371 // km
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)
	lat1Rad := lat1 * (math.Pi / 180.0)
	lat2Rad := lat2 * (math.Pi / 180.0)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}

// CreateProfileRequest represents profile creation request
type CreateProfileRequest struct {
	DisplayName       string   `json:"display_name" binding:"required,min=2,max=100"`
	Bio               *string  `json:"bio" binding:"omitempty,max=500"`
	City              *string  `json:"city" binding:"omitempty,max=100"`
	Interests         []string `json:"interests" binding:"omitempty,max=10"`
	PrefMinAge        *int     `json:"pref_min_age" binding:"omitempty,min=18,max=100"`
	PrefMaxAge        *int     `json:"pref_max_age" binding:"omitempty,min=18,max=100"`
	PrefMaxDistanceKm *int     `json:"pref_max_distance_km" binding:"omitempty,min=1,max=1000"`
}

// UpdateProfileRequest represents profile update request
type UpdateProfileRequest struct {
	DisplayName       *string   `json:"display_name" binding:"omitempty,min=2,max=100"`
	Bio               *string   `json:"bio" binding:"omitempty,max=500"`
	City              *string   `json:"city" binding:"omitempty,max=100"`
	Interests         *[]string `json:"interests" binding:"omitempty,max=10"`
	LocationLat       *float64  `json:"location_lat" binding:"omitempty,min=-90,max=90"`
	LocationLon       *float64  `json:"location_lon" binding:"omitempty,min=-180,max=180"`
	PrefMinAge        *int      `json:"pref_min_age" binding:"omitempty,min=18,max=100"`
	PrefMaxAge        *int      `json:"pref_max_age" binding:"omitempty,min=18,max=100"`
	PrefMaxDistanceKm *int      `json:"pref_max_distance_km" binding:"omitempty,min=1,max=1000"`
}

// ProfileResponse represents profile response with additional info
type ProfileResponse struct {
	*domain.Profile
	Age        int      `json:"age,omitempty"`
	DistanceKm *float64 `json:"distance_km,omitempty"`
}

// GetMyProfile returns current user's profile
func (uc *ProfileUseCase) GetMyProfile(ctx context.Context, userID int) (*domain.Profile, error) {
	profile, err := uc.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

// GetProfileByUserID returns profile by user ID with calculated age and distance
func (uc *ProfileUseCase) GetProfileByUserID(ctx context.Context, targetUserID int, currentUserID *int) (*ProfileResponse, error) {
	profile, err := uc.profileRepo.GetByUserID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}

	user, err := uc.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}

	response := &ProfileResponse{
		Profile: profile,
		Age:     user.Age(),
	}

	// Calculate distance if current user location is available
	if currentUserID != nil {
		currentProfile, err := uc.profileRepo.GetByUserID(ctx, *currentUserID)
		if err == nil && currentProfile.LocationLat != nil && currentProfile.LocationLon != nil &&
			profile.LocationLat != nil && profile.LocationLon != nil {
			distance := calculateDistance(
				*currentProfile.LocationLat, *currentProfile.LocationLon,
				*profile.LocationLat, *profile.LocationLon,
			)
			response.DistanceKm = &distance
		}
	}

	return response, nil
}

// CreateProfile creates a new profile (onboarding)
func (uc *ProfileUseCase) CreateProfile(ctx context.Context, userID int, req *CreateProfileRequest) (*domain.Profile, error) {
	// Check if profile already exists
	existingProfile, err := uc.profileRepo.GetByUserID(ctx, userID)
	if err == nil && existingProfile != nil {
		return nil, domain.ErrProfileAlreadyExists
	}

	profile := &domain.Profile{
		UserID:               userID,
		DisplayName:          req.DisplayName,
		Bio:                  req.Bio,
		City:                 req.City,
		Interests:            req.Interests,
		PrefMinAge:           req.PrefMinAge,
		PrefMaxAge:           req.PrefMaxAge,
		PrefMaxDistanceKm:    req.PrefMaxDistanceKm,
		IsOnboardingComplete: true,
	}

	if err := uc.profileRepo.Create(ctx, profile); err != nil {
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}

	return profile, nil
}

// UpdateProfile updates user profile
func (uc *ProfileUseCase) UpdateProfile(ctx context.Context, userID int, req *UpdateProfileRequest) (*domain.Profile, error) {
	profile, err := uc.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.DisplayName != nil {
		profile.DisplayName = *req.DisplayName
	}
	if req.Bio != nil {
		profile.Bio = req.Bio
	}
	if req.City != nil {
		profile.City = req.City
	}
	if req.Interests != nil {
		profile.Interests = *req.Interests
	}
	if req.LocationLat != nil {
		profile.LocationLat = req.LocationLat
	}
	if req.LocationLon != nil {
		profile.LocationLon = req.LocationLon
	}
	if req.PrefMinAge != nil {
		profile.PrefMinAge = req.PrefMinAge
	}
	if req.PrefMaxAge != nil {
		profile.PrefMaxAge = req.PrefMaxAge
	}
	if req.PrefMaxDistanceKm != nil {
		profile.PrefMaxDistanceKm = req.PrefMaxDistanceKm
	}

	if err := uc.profileRepo.Update(ctx, profile); err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	return profile, nil
}
