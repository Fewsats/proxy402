package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"linkshrink/internal/core/models"
	"linkshrink/store/sqlc"
	"linkshrink/users"
)

// Create inserts a new user in the database.
func (s *Store) CreateUser(ctx context.Context, user *models.User) (uint64, error) {
	userID, err := s.queries.CreateUser(ctx, sqlc.CreateUserParams{
		Email:    user.Email,
		Name:     user.Name,
		GoogleID: user.GoogleID,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	return uint64(userID), nil
}

// FindByID retrieves a user by ID.
func (s *Store) FindUserByID(ctx context.Context, id uint) (*models.User, error) {
	dbUser, err := s.queries.GetUserByID(ctx, int64(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, users.ErrUserNotFound
		}
		return nil, err
	}

	return convertToUserModel(dbUser), nil
}

// FindByGoogleID retrieves a user by Google ID.
func (s *Store) FindUserByGoogleID(ctx context.Context, googleID string) (*models.User, error) {
	dbUser, err := s.queries.GetUserByGoogleID(ctx, googleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, users.ErrUserNotFound
		}
		return nil, err
	}

	return convertToUserModel(dbUser), nil
}

// Helper function to convert sqlc User to models.User
func convertToUserModel(dbUser sqlc.User) *models.User {
	return &models.User{
		Email:     dbUser.Email,
		Name:      dbUser.Name,
		GoogleID:  dbUser.GoogleID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
	}
}
