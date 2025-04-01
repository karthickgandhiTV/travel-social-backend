package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/karthickgandhiTV/travel-social-backend/internal/config"
	_ "github.com/lib/pq"
)

// DB is a wrapper around sql.DB
type DB struct {
	*sql.DB
}

// New creates a new database connection
func New(cfg *config.Config) (*DB, error) {
	db, err := sql.Open("postgres", cfg.GetDBConnString())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Check the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Connected to database")

	if err := initSchema(db); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &DB{DB: db}, nil
}

// initSchema creates the necessary tables if they don't exist
func initSchema(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(36) PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			first_name VARCHAR(255),
			last_name VARCHAR(255),
			profile_picture TEXT,
			bio TEXT,
			interests TEXT[],
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS travel_preferences (
			id VARCHAR(36) PRIMARY KEY,
			user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			preferred_activities TEXT[],
			travel_style VARCHAR(255),
			languages_spoken TEXT[],
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		)`,
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			return fmt.Errorf("error executing query %s: %w", query, err)
		}
	}

	return nil
}
