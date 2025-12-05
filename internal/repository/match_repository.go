package repository

import (
	"context"

	"github.com/gdugdh24/mpit2026-backend/internal/domain"
)

type MatchRepository interface {
	Create(ctx context.Context, match *domain.Match) error
	GetByID(ctx context.Context, id int) (*domain.Match, error)
	GetByUsers(ctx context.Context, user1ID, user2ID int) (*domain.Match, error)
	GetUserMatches(ctx context.Context, userID int, limit, offset int) ([]*domain.Match, error)
	GetActiveMatches(ctx context.Context, userID int) ([]*domain.Match, error)
	UpdateStatus(ctx context.Context, id int, isActive bool) error
	Delete(ctx context.Context, id int) error
	UpdateAIFields(ctx context.Context, matchID int, explanation string, icebreakers []string) error
}
