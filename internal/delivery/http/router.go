package http

import (
	"github.com/gdugdh24/mpit2026-backend/internal/delivery/http/handler"
	"github.com/gdugdh24/mpit2026-backend/internal/delivery/http/middleware"
	"github.com/gin-gonic/gin"
)

type Router struct {
	authHandler    *handler.AuthHandler
	profileHandler *handler.ProfileHandler
	bigFiveHandler *handler.BigFiveHandler
	feedHandler    *handler.FeedHandler
	swipeHandler   *handler.SwipeHandler
	authMiddleware *middleware.AuthMiddleware
}

func NewRouter(
	authHandler *handler.AuthHandler,
	profileHandler *handler.ProfileHandler,
	bigFiveHandler *handler.BigFiveHandler,
	feedHandler *handler.FeedHandler,
	swipeHandler *handler.SwipeHandler,
	authMiddleware *middleware.AuthMiddleware,
) *Router {
	return &Router{
		authHandler:    authHandler,
		profileHandler: profileHandler,
		bigFiveHandler: bigFiveHandler,
		feedHandler:    feedHandler,
		swipeHandler:   swipeHandler,
		authMiddleware: authMiddleware,
	}
}

func (r *Router) Setup() *gin.Engine {
	router := gin.Default()

	// Health check (supports both GET and HEAD)
	healthHandler := func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	}
	router.GET("/health", healthHandler)
	router.HEAD("/health", healthHandler)

	// API v1
	v1 := router.Group("/api/v1")
	{
		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/vk", r.authHandler.VKAuth)
			auth.POST("/logout", r.authMiddleware.RequireAuth(), r.authHandler.Logout)
			auth.GET("/me", r.authMiddleware.RequireAuth(), r.authHandler.Me)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(r.authMiddleware.RequireAuth())
		{
			// Profile routes
			profile := protected.Group("/profile")
			{
				profile.GET("/me", r.profileHandler.GetMyProfile)
				profile.PUT("/me", r.profileHandler.UpdateMyProfile)
				profile.POST("/complete-onboarding", r.profileHandler.CompleteOnboarding)
				profile.GET("/:user_id", r.profileHandler.GetProfileByUserID)
			}

			// Big Five routes
			bigFive := protected.Group("/big-five")
			{
				bigFive.POST("/submit", r.bigFiveHandler.SubmitAnswers)
				bigFive.GET("/my-results", r.bigFiveHandler.GetMyResults)
				bigFive.GET("/user/:user_id", r.bigFiveHandler.GetUserResults)
			}

			// Feed routes
			feed := protected.Group("/feed")
			{
				feed.GET("/next", r.feedHandler.GetNextUser)
				feed.POST("/reset-dislikes", r.feedHandler.ResetDislikes)
			}

			// Swipe routes
			swipe := protected.Group("/swipe")
			{
				swipe.POST("", r.swipeHandler.CreateSwipe)
				swipe.GET("/likes-received", r.swipeHandler.GetLikesReceived)
			}

			// TODO: Add match routes
			// TODO: Add message routes
			// TODO: Add notification routes
			// TODO: Add dashboard /me route
		}

		// Big Five questions (public)
		v1.GET("/big-five/questions", r.bigFiveHandler.GetQuestions)
	}

	return router
}
