package domain

import "time"

type Profile struct {
	ID                   int        `json:"id" db:"id"`
	UserID               int        `json:"user_id" db:"user_id"`
	DisplayName          string     `json:"display_name" db:"display_name"`
	Bio                  *string    `json:"bio" db:"bio"`
	City                 *string    `json:"city" db:"city"`
	Interests            []string   `json:"interests" db:"interests"`
	LocationLat          *float64   `json:"location_lat" db:"location_lat"`
	LocationLon          *float64   `json:"location_lon" db:"location_lon"`
	LocationUpdatedAt    *time.Time `json:"location_updated_at" db:"location_updated_at"`
	PrefMinAge           *int       `json:"pref_min_age" db:"pref_min_age"`
	PrefMaxAge           *int       `json:"pref_max_age" db:"pref_max_age"`
	PrefMaxDistanceKm    *int       `json:"pref_max_distance_km" db:"pref_max_distance_km"`
	PrefOpenness          *float64   `json:"pref_openness" db:"pref_openness"`
	PrefConscientiousness *float64   `json:"pref_conscientiousness" db:"pref_conscientiousness"`
	PrefExtraversion      *float64   `json:"pref_extraversion" db:"pref_extraversion"`
	PrefAgreeableness     *float64   `json:"pref_agreeableness" db:"pref_agreeableness"`
	PrefNeuroticism       *float64   `json:"pref_neuroticism" db:"pref_neuroticism"`
	IsOnboardingComplete bool       `json:"is_onboarding_complete" db:"is_onboarding_complete"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
}
