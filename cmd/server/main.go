package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gdugdh24/mpit2026-backend/internal/config"
	"github.com/gdugdh24/mpit2026-backend/internal/infrastructure/container"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize dependency injection container
	app, err := container.NewContainer(cfg)
	if err != nil {
		fmt.Printf("Failed to initialize application: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := app.Close(); err != nil {
			fmt.Printf("Error closing application: %v\n", err)
		}
	}()

	// Channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		if err := app.Server.Start(); err != nil {
			fmt.Printf("Server error: %v\n", err)
			quit <- syscall.SIGTERM
		}
	}()

	fmt.Printf("Server started successfully on %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Println("Press Ctrl+C to stop")

	// Wait for interrupt signal
	<-quit

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10)
	defer cancel()

	if err := app.Server.Shutdown(ctx); err != nil {
		fmt.Printf("Server shutdown error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Server exited properly")
}
