package user

import (
	"context"
	"fmt"

	"github.com/karthickgandhiTV/travel-social-backend/internal/config"
	"github.com/karthickgandhiTV/travel-social-backend/internal/graph/models"
	ory "github.com/ory/client-go"
	kratosclient "github.com/ory/kratos-client-go"
)

type Service struct {
	repo   *Repository
	config *config.Config
}

func NewService(repo *Repository, cfg *config.Config) *Service {
	return &Service{
		repo:   repo,
		config: cfg,
	}
}

func (s *Service) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	return s.repo.GetUserByID(ctx, id)
}

func (s *Service) GetOrCreateUser(ctx context.Context, id string) (*models.User, error) {
	// Try to get existing user
	user, err := s.repo.GetUserByID(ctx, id)
	if err == nil {
		return user, nil
	}

	// User not found, get email from Kratos and create user
	email, err := s.getUserEmailFromKratos(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info from Kratos: %w", err)
	}

	return s.repo.CreateUser(ctx, id, email)
}

func (s *Service) getUserEmailFromKratos(ctx context.Context, id string) (string, error) {
	client := ory.NewAPIClient(&ory.Configuration{
		Servers: []ory.ServerConfiguration{
			{
				URL: s.config.KratosAdminURL,
			},
		},
	})

	identity, _, err := client.IdentityAPI.GetIdentity(ctx, id).Execute()
	if err != nil {
		return "", fmt.Errorf("failed to get identity from Kratos: %w", err)
	}

	// Extract email from traits
	traits, ok := identity.Traits.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid traits format")
	}

	email, ok := traits["email"].(string)
	if !ok {
		return "", fmt.Errorf("email not found in traits")
	}

	return email, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, input models.UpdateProfileInput) (*models.User, error) {
	return s.repo.UpdateProfile(ctx, userID, input)
}

func (s *Service) GetTravelPreferences(ctx context.Context, userID string) (*models.TravelPreferences, error) {
	return s.repo.GetTravelPreferences(ctx, userID)
}

func (s *Service) UpdateTravelPreferences(ctx context.Context, userID string, input models.UpdateTravelPreferencesInput) (*models.TravelPreferences, error) {
	return s.repo.UpdateTravelPreferences(ctx, userID, input)
}

func (s *Service) SearchUsers(ctx context.Context, query string) ([]*models.User, error) {
	return s.repo.SearchUsers(ctx, query)
}

func (s *Service) GetUserSession(ctx context.Context, sessionToken string) (*kratosclient.Session, error) {
	client := kratosclient.NewAPIClient(&kratosclient.Configuration{
		Servers: []kratosclient.ServerConfiguration{
			{
				URL: s.config.KratosPublicURL,
			},
		},
	})

	resp, r, err := client.FrontendAPI.ToSession(ctx).
		Cookie("ory_kratos_session=" + sessionToken).
		Execute()

	if err != nil || r.StatusCode != 200 {
		return nil, fmt.Errorf("invalid session: %w", err)
	}

	return resp, nil
}
