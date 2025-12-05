package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gdugdh24/mpit2026-backend/internal/domain"
	"github.com/gdugdh24/mpit2026-backend/internal/repository"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type profileRepository struct {
	db *sqlx.DB
}

func NewProfileRepository(db *sqlx.DB) repository.ProfileRepository {
	return &profileRepository{db: db}
}

func (r *profileRepository) Create(ctx context.Context, profile *domain.Profile) error {
	query := `
		INSERT INTO profiles (
			user_id, display_name, bio, city, interests,
			location_lat, location_lon, location_updated_at,
			pref_min_age, pref_max_age, pref_max_distance_km, is_onboarding_complete,
			pref_openness, pref_conscientiousness, pref_extraversion, pref_agreeableness, pref_neuroticism
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(
		ctx, query,
		profile.UserID, profile.DisplayName, profile.Bio, profile.City,
		pq.Array(profile.Interests), profile.LocationLat, profile.LocationLon,
		profile.LocationUpdatedAt, profile.PrefMinAge, profile.PrefMaxAge,
		profile.PrefMaxDistanceKm, profile.IsOnboardingComplete,
		profile.PrefOpenness, profile.PrefConscientiousness, profile.PrefExtraversion,
		profile.PrefAgreeableness, profile.PrefNeuroticism,
	).Scan(&profile.ID, &profile.CreatedAt, &profile.UpdatedAt)
}

func (r *profileRepository) GetByID(ctx context.Context, id int) (*domain.Profile, error) {
	var profile domain.Profile
	query := `SELECT * FROM profiles WHERE id = $1`
	err := r.db.GetContext(ctx, &profile, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrProfileNotFound
		}
		return nil, err
	}
	return &profile, nil
}

func (r *profileRepository) GetByUserID(ctx context.Context, userID int) (*domain.Profile, error) {
	var profile domain.Profile
	query := `
		SELECT id, user_id, display_name, bio, city, interests,
		       location_lat, location_lon, location_updated_at,
		       pref_min_age, pref_max_age, pref_max_distance_km,
		       is_onboarding_complete,
		       pref_openness, pref_conscientiousness, pref_extraversion,
		       pref_agreeableness, pref_neuroticism,
		       created_at, updated_at
		FROM profiles WHERE user_id = $1
	`
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&profile.ID, &profile.UserID, &profile.DisplayName, &profile.Bio, &profile.City, pq.Array(&profile.Interests),
		&profile.LocationLat, &profile.LocationLon, &profile.LocationUpdatedAt,
		&profile.PrefMinAge, &profile.PrefMaxAge, &profile.PrefMaxDistanceKm,
		&profile.IsOnboardingComplete,
		&profile.PrefOpenness, &profile.PrefConscientiousness, &profile.PrefExtraversion,
		&profile.PrefAgreeableness, &profile.PrefNeuroticism,
		&profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrProfileNotFound
		}
		return nil, err
	}
	return &profile, nil
}

func (r *profileRepository) Update(ctx context.Context, profile *domain.Profile) error {
	query := `
		UPDATE profiles
		SET display_name = $1, bio = $2, city = $3, interests = $4,
		    location_lat = $5, location_lon = $6, location_updated_at = $7,
		    pref_min_age = $8, pref_max_age = $9, pref_max_distance_km = $10,
		    is_onboarding_complete = $11,
			pref_openness = $12, pref_conscientiousness = $13, pref_extraversion = $14,
			pref_agreeableness = $15, pref_neuroticism = $16,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $17
		RETURNING updated_at
	`
	return r.db.QueryRowContext(
		ctx, query,
		profile.DisplayName, profile.Bio, profile.City, pq.Array(profile.Interests),
		profile.LocationLat, profile.LocationLon, profile.LocationUpdatedAt,
		profile.PrefMinAge, profile.PrefMaxAge, profile.PrefMaxDistanceKm,
		profile.IsOnboardingComplete,
		profile.PrefOpenness, profile.PrefConscientiousness, profile.PrefExtraversion,
		profile.PrefAgreeableness, profile.PrefNeuroticism,
		profile.ID,
	).Scan(&profile.UpdatedAt)
}

func (r *profileRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM profiles WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrProfileNotFound
	}
	return nil
}

func (r *profileRepository) UpdateOnboardingStatus(ctx context.Context, userID int, isComplete bool) error {
	query := `
		UPDATE profiles
		SET is_onboarding_complete = $1, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $2
	`
	result, err := r.db.ExecContext(ctx, query, isComplete, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrProfileNotFound
	}
	return nil
}

func (r *profileRepository) SearchProfiles(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.Profile, error) {
	var profiles []*domain.Profile

	query := `SELECT * FROM profiles WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	if city, ok := filters["city"].(string); ok && city != "" {
		query += fmt.Sprintf(" AND city = $%d", argCount)
		args = append(args, city)
		argCount++
	}

	if interests, ok := filters["interests"].([]string); ok && len(interests) > 0 {
		query += fmt.Sprintf(" AND interests && $%d", argCount)
		args = append(args, pq.Array(interests))
		argCount++
	}

	if onboardingComplete, ok := filters["is_onboarding_complete"].(bool); ok {
		query += fmt.Sprintf(" AND is_onboarding_complete = $%d", argCount)
		args = append(args, onboardingComplete)
		argCount++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	err := r.db.SelectContext(ctx, &profiles, query, args...)
	return profiles, err
}
