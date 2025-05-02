package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"linkshrink/internal/core/models"
	"linkshrink/users"
)

// UserService provides user-related business logic.
type UserService struct {
	logger *slog.Logger
	store  users.Store
}

// NewUserService creates a new UserService.
func NewUserService(logger *slog.Logger, store users.Store) *UserService {
	return &UserService{logger: logger, store: store}
}

// FindOrCreateUser finds a user by Google ID or creates a new one.
func (s *UserService) FindOrCreateUser(ctx context.Context, email, name, googleID string) (*models.User, error) {
	// Try to find user by Google ID
	user, err := s.store.FindUserByGoogleID(ctx, googleID)
	if err == nil {
		// User found, return it
		return user, nil
	} else if !errors.Is(err, users.ErrUserNotFound) {
		// A different error occurred during lookup
		return nil, fmt.Errorf("error checking Google user: %w", err)
	}

	// User not found, create a new one
	newUser := &models.User{
		Email:    email,
		Name:     name,
		GoogleID: googleID,
	}

	id, err := s.store.CreateUser(ctx, newUser)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	newUser.ID = uint(id)

	return newUser, nil
}

// GetUserByID retrieves a user by their ID.
func (s *UserService) GetUserByID(ctx context.Context, userID uint) (*models.User, error) {
	user, err := s.store.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			return nil, users.ErrUserNotFound
		}
		return nil, fmt.Errorf("error finding user: %w", err)
	}
	return user, nil
}
