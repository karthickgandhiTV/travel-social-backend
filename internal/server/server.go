package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/karthickgandhiTV/travel-social-backend/internal/auth"
	"github.com/karthickgandhiTV/travel-social-backend/internal/config"
	"github.com/karthickgandhiTV/travel-social-backend/internal/db"
	"github.com/karthickgandhiTV/travel-social-backend/internal/graph"
	"github.com/karthickgandhiTV/travel-social-backend/internal/graph/generated"
	"github.com/karthickgandhiTV/travel-social-backend/internal/user"
)

// Server represents the HTTP server
type Server struct {
	router chi.Router
	config *config.Config
}

// New creates a new server instance
func New(cfg *config.Config) (*Server, error) {
	// Connect to database
	database, err := db.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Set up repositories and services
	userRepo := user.NewRepository(database)
	userService := user.NewService(userRepo, cfg)

	// Set up router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(auth.Middleware(cfg))

	// Set up GraphQL handler
	resolver := &graph.Resolver{
		UserService: userService,
	}

	gqlServer := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))

	// Routes
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	r.Handle("/playground", playground.Handler("GraphQL playground", "/query"))
	r.Handle("/query", gqlServer)

	return &Server{
		router: r,
		config: cfg,
	}, nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	port := s.config.AppPort
	addr := fmt.Sprintf(":%s", port)

	log.Printf("Server is running on http://localhost:%s", port)
	log.Printf("GraphQL playground available at http://localhost:%s/playground", port)

	return http.ListenAndServe(addr, s.router)
}
