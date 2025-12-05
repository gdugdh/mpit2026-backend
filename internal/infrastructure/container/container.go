package container

import (
	"fmt"

	"github.com/gdugdh24/mpit2026-backend/internal/config"
	"github.com/gdugdh24/mpit2026-backend/internal/delivery/http"
	"github.com/gdugdh24/mpit2026-backend/internal/delivery/http/handler"
	"github.com/gdugdh24/mpit2026-backend/internal/delivery/http/middleware"
	"github.com/gdugdh24/mpit2026-backend/internal/infrastructure/database"
	"github.com/gdugdh24/mpit2026-backend/internal/infrastructure/gemini"
	"github.com/gdugdh24/mpit2026-backend/internal/infrastructure/server"
	"github.com/gdugdh24/mpit2026-backend/internal/repository/postgres"
	"github.com/gdugdh24/mpit2026-backend/internal/usecase/auth"
	"github.com/gdugdh24/mpit2026-backend/internal/usecase/bigfive"
	"github.com/gdugdh24/mpit2026-backend/internal/usecase/feed"
	"github.com/gdugdh24/mpit2026-backend/internal/usecase/profile"
	"github.com/gdugdh24/mpit2026-backend/internal/usecase/swipe"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

// Container holds all application dependencies
type Container struct {
	Config *config.Config
	DB     *sqlx.DB
	Redis  *redis.Client
	Server *server.Server
	Gemini *gemini.GeminiClient
}

// NewContainer creates a new dependency injection container
func NewContainer(cfg *config.Config) (*Container, error) {
	// Initialize database
	db, err := database.NewPostgresDB(&cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// // Initialize Redis
	// redisClient, err := database.NewRedisClient(&cfg.Redis)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to initialize redis: %w", err)
	// }

	// Initialize Gemini Client
	geminiClient, err := gemini.NewGeminiClient(cfg.GeminiAPIKey)
	if err != nil {
		fmt.Printf("Warning: Failed to initialize Gemini client: %v\n", err)
		// Don't fail, just continue without AI features
	}

	// Initialize repositories
	userRepo := postgres.NewUserRepository(db)
	profileRepo := postgres.NewProfileRepository(db)
	sessionRepo := postgres.NewSessionRepository(db)
	swipeRepo := postgres.NewSwipeRepository(db)
	matchRepo := postgres.NewMatchRepository(db)
	// messageRepo := postgres.NewMessageRepository(db)
	// notificationRepo := postgres.NewNotificationRepository(db)
	bigFiveRepo := postgres.NewBigFiveRepository(db)

	// Initialize use cases
	authUseCase := auth.NewVKAuthUseCase(
		userRepo,
		profileRepo,
		sessionRepo,
		cfg.VK.SecretKey,
		cfg.JWT.AccessSecret,
	)

	profileUseCase := profile.NewProfileUseCase(
		profileRepo,
		userRepo,
		geminiClient,
	)

	bigFiveUseCase := bigfive.NewBigFiveUseCase(
		bigFiveRepo,
	)

	feedUseCase := feed.NewFeedUseCase(
		userRepo,
		profileRepo,
		swipeRepo,
	)

	swipeUseCase := swipe.NewSwipeUseCase(
		swipeRepo,
		matchRepo,
		profileRepo,
		userRepo,
		geminiClient,
	)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authUseCase)
	profileHandler := handler.NewProfileHandler(profileUseCase)
	bigFiveHandler := handler.NewBigFiveHandler(bigFiveUseCase)
	feedHandler := handler.NewFeedHandler(feedUseCase)
	swipeHandler := handler.NewSwipeHandler(swipeUseCase)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authUseCase)

	// Initialize router
	router := http.NewRouter(
		authHandler,
		profileHandler,
		bigFiveHandler,
		feedHandler,
		swipeHandler,
		authMiddleware,
	)

	// Setup routes
	ginRouter := router.Setup()

	// Initialize server
	srv := server.NewServer(&cfg.Server, ginRouter)

	return &Container{
		Config: cfg,
		DB:     db,
		Redis:  nil,
		Server: srv,
		Gemini: geminiClient,
	}, nil
}

// Close closes all connections
func (c *Container) Close() error {
	// Close Redis
	if c.Redis != nil {
		if err := c.Redis.Close(); err != nil {
			fmt.Printf("Error closing Redis: %v\n", err)
		}
	}

	// Close database
	if c.DB != nil {
		if err := c.DB.Close(); err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
	}

	return nil
}
