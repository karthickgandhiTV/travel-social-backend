package main

import (
	"log"
	"os"

	"github.com/karthickgandhiTV/travel-social-backend/internal/config"
	"github.com/karthickgandhiTV/travel-social-backend/internal/server"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Create and start server
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
		os.Exit(1)
	}

	// Start the server
	log.Println("Starting server...")
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
		os.Exit(1)
	}
}
