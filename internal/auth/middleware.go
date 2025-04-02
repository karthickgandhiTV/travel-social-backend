package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/karthickgandhiTV/travel-social-backend/internal/config"
	kratosclient "github.com/ory/kratos-client-go"
)

type contextKey string

const (
	userIDKey contextKey = "userID"
	userKey   contextKey = "user"
)

// Middleware validates the session and sets user info in context
func Middleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for certain paths like health check
			if r.URL.Path == "/health" || r.URL.Path == "/playground" {
				next.ServeHTTP(w, r)
				return
			}

			// Get session cookie
			cookie, err := r.Cookie("ory_kratos_session")
			if err != nil {
				// No session cookie found, check if this is an API call with bearer token
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" {
					// No authentication provided, let the resolver handle unauthorized access
					next.ServeHTTP(w, r)
					return
				}

				// Extract bearer token
				parts := strings.Split(authHeader, " ")
				if len(parts) != 2 || parts[0] != "Bearer" {
					http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
					return
				}

				// Use the token to validate with Kratos
				sessionToken := parts[1]
				userInfo, err := validateSession(cfg, sessionToken)
				if err != nil {
					http.Error(w, "Invalid or expired session", http.StatusUnauthorized)
					return
				}

				// Set user info in context
				ctx := context.WithValue(r.Context(), userIDKey, userInfo.Identity.Id)
				ctx = context.WithValue(ctx, userKey, userInfo)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Validate session using cookie
			userInfo, err := validateSession(cfg, cookie.Value)
			if err != nil {
				http.Error(w, "Invalid or expired session", http.StatusUnauthorized)
				return
			}

			// Set user info in context
			ctx := context.WithValue(r.Context(), userIDKey, userInfo.Identity.Id)
			ctx = context.WithValue(ctx, userKey, userInfo)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// validateSession checks if the session is valid with Kratos
func validateSession(cfg *config.Config, sessionToken string) (*kratosclient.Session, error) {
	client := kratosclient.NewAPIClient(&kratosclient.Configuration{
		Servers: []kratosclient.ServerConfiguration{
			{
				URL: cfg.KratosPublicURL,
			},
		},
	})

	resp, r, err := client.FrontendAPI.ToSession(context.Background()).
		Cookie("ory_kratos_session=" + sessionToken).
		Execute()

	if err != nil || r.StatusCode != 200 {
		return nil, errors.New("invalid session")
	}

	return resp, nil
}

// GetUserIDFromContext retrieves the user ID from context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok
}

// GetUserFromContext retrieves the user info from context
func GetUserFromContext(ctx context.Context) (*kratosclient.Session, bool) {
	user, ok := ctx.Value(userKey).(*kratosclient.Session)
	return user, ok
}

// RequireAuth checks if the user is authenticated
func RequireAuth(ctx context.Context) (string, error) {
	userID, ok := GetUserIDFromContext(ctx)
	if !ok || userID == "" {
		return "", errors.New("not authenticated")
	}
	return userID, nil
}
