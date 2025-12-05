package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/gdugdh24/mpit2026-backend/internal/domain"
	"github.com/gdugdh24/mpit2026-backend/internal/repository"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type matchRepository struct {
	db *sqlx.DB
}

func NewMatchRepository(db *sqlx.DB) repository.MatchRepository {
	return &matchRepository{db: db}
}

func (r *matchRepository) Create(ctx context.Context, match *domain.Match) error {
	// Ensure user1_id < user2_id for constraint
	user1ID, user2ID := match.User1ID, match.User2ID
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	query := `
		INSERT INTO matches (user1_id, user2_id, is_active, match_explanation, icebreakers)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	err := r.db.QueryRowContext(ctx, query, user1ID, user2ID, match.IsActive, match.Explanation, pq.Array(match.Icebreakers)).
		Scan(&match.ID, &match.CreatedAt)

	match.User1ID = user1ID
	match.User2ID = user2ID
	return err
}

func (r *matchRepository) GetByID(ctx context.Context, id int) (*domain.Match, error) {
	var match domain.Match
	query := `SELECT * FROM matches WHERE id = $1`
	err := r.db.GetContext(ctx, &match, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrMatchNotFound
		}
		return nil, err
	}
	return &match, nil
}

func (r *matchRepository) GetByUsers(ctx context.Context, user1ID, user2ID int) (*domain.Match, error) {
	// Ensure user1_id < user2_id
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	var match domain.Match
	query := `SELECT * FROM matches WHERE user1_id = $1 AND user2_id = $2`
	err := r.db.GetContext(ctx, &match, query, user1ID, user2ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrMatchNotFound
		}
		return nil, err
	}
	return &match, nil
}

func (r *matchRepository) GetUserMatches(ctx context.Context, userID int, limit, offset int) ([]*domain.Match, error) {
	var matches []*domain.Match
	query := `
		SELECT * FROM matches
		WHERE (user1_id = $1 OR user2_id = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	err := r.db.SelectContext(ctx, &matches, query, userID, limit, offset)
	return matches, err
}

func (r *matchRepository) GetActiveMatches(ctx context.Context, userID int) ([]*domain.Match, error) {
	var matches []*domain.Match
	query := `
		SELECT * FROM matches
		WHERE (user1_id = $1 OR user2_id = $1) AND is_active = true
		ORDER BY created_at DESC
	`
	err := r.db.SelectContext(ctx, &matches, query, userID)
	return matches, err
}

func (r *matchRepository) UpdateStatus(ctx context.Context, id int, isActive bool) error {
	query := `UPDATE matches SET is_active = $1 WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, isActive, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrMatchNotFound
	}
	return nil
}

func (r *matchRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM matches WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrMatchNotFound
	}
	return nil
}
