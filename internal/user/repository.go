package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/karthickgandhiTV/travel-social-backend/internal/db"
	"github.com/karthickgandhiTV/travel-social-backend/internal/graph/models"
	"github.com/lib/pq"
)

type Repository struct {
	db *db.DB
}

func NewRepository(db *db.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, email, first_name, last_name, profile_picture, bio, interests, 
		       created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	var firstName, lastName, profilePicture, bio sql.NullString
	var interests []sql.NullString
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &firstName, &lastName, &profilePicture, &bio,
		pq.Array(&interests), &createdAt, &updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("error querying user: %w", err)
	}

	// Convert null strings to pointers
	if firstName.Valid {
		user.FirstName = &firstName.String
	}
	if lastName.Valid {
		user.LastName = &lastName.String
	}
	if profilePicture.Valid {
		user.ProfilePicture = &profilePicture.String
	}
	if bio.Valid {
		user.Bio = &bio.String
	}

	// Convert sql.NullString array to string array
	for _, i := range interests {
		if i.Valid {
			user.Interests = append(user.Interests, i.String)
		}
	}

	user.CreatedAt = createdAt.Format(time.RFC3339)
	user.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &user, nil
}

func (r *Repository) CreateUser(ctx context.Context, id, email string) (*models.User, error) {
	query := `
		INSERT INTO users (id, email)
		VALUES ($1, $2)
		RETURNING id, email, created_at, updated_at
	`

	var user models.User
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, id, email).Scan(
		&user.ID, &user.Email, &createdAt, &updatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("error creating user: %w", err)
	}

	user.CreatedAt = createdAt.Format(time.RFC3339)
	user.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &user, nil
}

func (r *Repository) UpdateProfile(ctx context.Context, userID string, input models.UpdateProfileInput) (*models.User, error) {
	query := `
		UPDATE users
		SET 
			first_name = COALESCE($2, first_name),
			last_name = COALESCE($3, last_name),
			profile_picture = COALESCE($4, profile_picture),
			bio = COALESCE($5, bio),
			interests = CASE WHEN $6::text[] IS NOT NULL THEN $6::text[] ELSE interests END,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, email, first_name, last_name, profile_picture, bio, interests, created_at, updated_at
	`

	var user models.User
	var firstName, lastName, profilePicture, bio sql.NullString
	var interests []sql.NullString
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, userID, input.FirstName, input.LastName,
		input.ProfilePicture, input.Bio, pq.Array(input.Interests)).Scan(
		&user.ID, &user.Email, &firstName, &lastName, &profilePicture, &bio,
		pq.Array(&interests), &createdAt, &updatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("error updating profile: %w", err)
	}

	// Convert null strings to pointers
	if firstName.Valid {
		user.FirstName = &firstName.String
	}
	if lastName.Valid {
		user.LastName = &lastName.String
	}
	if profilePicture.Valid {
		user.ProfilePicture = &profilePicture.String
	}
	if bio.Valid {
		user.Bio = &bio.String
	}

	// Convert sql.NullString array to string array
	for _, i := range interests {
		if i.Valid {
			user.Interests = append(user.Interests, i.String)
		}
	}

	user.CreatedAt = createdAt.Format(time.RFC3339)
	user.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &user, nil
}

func (r *Repository) GetTravelPreferences(ctx context.Context, userID string) (*models.TravelPreferences, error) {
	query := `
		SELECT id, user_id, preferred_activities, travel_style, languages_spoken, updated_at
		FROM travel_preferences
		WHERE user_id = $1
	`

	var prefs models.TravelPreferences
	var travelStyle sql.NullString
	var preferredActivities, languagesSpoken []sql.NullString
	var updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&prefs.ID, &prefs.UserID, pq.Array(&preferredActivities), &travelStyle,
		pq.Array(&languagesSpoken), &updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No preferences set yet
		}
		return nil, fmt.Errorf("error querying travel preferences: %w", err)
	}

	// Convert null strings to pointers
	if travelStyle.Valid {
		prefs.TravelStyle = &travelStyle.String
	}

	// Convert sql.NullString arrays to string arrays
	for _, a := range preferredActivities {
		if a.Valid {
			prefs.PreferredActivities = append(prefs.PreferredActivities, a.String)
		}
	}

	for _, l := range languagesSpoken {
		if l.Valid {
			prefs.LanguagesSpoken = append(prefs.LanguagesSpoken, l.String)
		}
	}

	prefs.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &prefs, nil
}

func (r *Repository) UpdateTravelPreferences(ctx context.Context, userID string, input models.UpdateTravelPreferencesInput) (*models.TravelPreferences, error) {
	// First check if travel preferences exist for this user
	var exists bool
	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM travel_preferences WHERE user_id = $1)", userID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("error checking travel preferences: %w", err)
	}

	var query string
	var params []interface{}

	if !exists {
		// Create new travel preferences
		query = `
			INSERT INTO travel_preferences (id, user_id, preferred_activities, travel_style, languages_spoken)
			VALUES (gen_random_uuid(), $1, $2, $3, $4)
			RETURNING id, user_id, preferred_activities, travel_style, languages_spoken, updated_at
		`
		params = []interface{}{userID, pq.Array(input.PreferredActivities), input.TravelStyle, pq.Array(input.LanguagesSpoken)}
	} else {
		// Update existing travel preferences
		query = `
			UPDATE travel_preferences
			SET 
				preferred_activities = CASE WHEN $2::text[] IS NOT NULL THEN $2::text[] ELSE preferred_activities END,
				travel_style = COALESCE($3, travel_style),
				languages_spoken = CASE WHEN $4::text[] IS NOT NULL THEN $4::text[] ELSE languages_spoken END,
				updated_at = NOW()
			WHERE user_id = $1
			RETURNING id, user_id, preferred_activities, travel_style, languages_spoken, updated_at
		`
		params = []interface{}{userID, pq.Array(input.PreferredActivities), input.TravelStyle, pq.Array(input.LanguagesSpoken)}
	}

	var prefs models.TravelPreferences
	var travelStyle sql.NullString
	var preferredActivities, languagesSpoken []sql.NullString
	var updatedAt time.Time

	err = r.db.QueryRowContext(ctx, query, params...).Scan(
		&prefs.ID, &prefs.UserID, pq.Array(&preferredActivities), &travelStyle,
		pq.Array(&languagesSpoken), &updatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("error updating travel preferences: %w", err)
	}

	// Convert null strings to pointers
	if travelStyle.Valid {
		prefs.TravelStyle = &travelStyle.String
	}

	// Convert sql.NullString arrays to string arrays
	for _, a := range preferredActivities {
		if a.Valid {
			prefs.PreferredActivities = append(prefs.PreferredActivities, a.String)
		}
	}

	for _, l := range languagesSpoken {
		if l.Valid {
			prefs.LanguagesSpoken = append(prefs.LanguagesSpoken, l.String)
		}
	}

	prefs.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &prefs, nil
}

func (r *Repository) SearchUsers(ctx context.Context, query string) ([]*models.User, error) {
	sqlQuery := `
		SELECT id, email, first_name, last_name, profile_picture, bio, interests, created_at, updated_at
		FROM users
		WHERE 
			LOWER(email) LIKE LOWER($1) OR
			LOWER(COALESCE(first_name, '')) LIKE LOWER($1) OR
			LOWER(COALESCE(last_name, '')) LIKE LOWER($1) OR
			LOWER(COALESCE(bio, '')) LIKE LOWER($1)
		LIMIT 20
	`

	rows, err := r.db.QueryContext(ctx, sqlQuery, "%"+query+"%")
	if err != nil {
		return nil, fmt.Errorf("error searching users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		var firstName, lastName, profilePicture, bio sql.NullString
		var interests []sql.NullString
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&user.ID, &user.Email, &firstName, &lastName, &profilePicture, &bio,
			pq.Array(&interests), &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning user row: %w", err)
		}

		// Convert null strings to pointers
		if firstName.Valid {
			user.FirstName = &firstName.String
		}
		if lastName.Valid {
			user.LastName = &lastName.String
		}
		if profilePicture.Valid {
			user.ProfilePicture = &profilePicture.String
		}
		if bio.Valid {
			user.Bio = &bio.String
		}

		// Convert sql.NullString array to string array
		for _, i := range interests {
			if i.Valid {
				user.Interests = append(user.Interests, i.String)
			}
		}

		user.CreatedAt = createdAt.Format(time.RFC3339)
		user.UpdatedAt = updatedAt.Format(time.RFC3339)

		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}
