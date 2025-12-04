package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gdugdh24/mpit2026-backend/internal/config"
	"github.com/gin-gonic/gin"
)

// Server represents HTTP server
type Server struct {
	httpServer *http.Server
	config     *config.ServerConfig
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.ServerConfig, router *gin.Engine) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:           fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			Handler:        router,
			ReadTimeout:    cfg.ReadTimeout,
			WriteTimeout:   cfg.WriteTimeout,
			MaxHeaderBytes: 1 << 20, // 1 MB
		},
		config: cfg,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	fmt.Printf("Starting server on %s:%d\n", s.config.Host, s.config.Port)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	fmt.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	fmt.Println("Server stopped")
	return nil
}
