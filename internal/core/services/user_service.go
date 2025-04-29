package services

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"linkshrink/internal/core/models"
	"linkshrink/internal/store"
)

// UserService provides user-related business logic.
type UserService struct {
	userStore *store.UserStore
}

// NewUserService creates a new UserService.
func NewUserService(userStore *store.UserStore) *UserService {
	return &UserService{userStore: userStore}
}

// FindOrCreateUser finds a user by Google ID or creates a new one.
func (s *UserService) FindOrCreateUser(email, name, googleID string) (*models.User, error) {
	// Try to find user by Google ID
	user, err := s.userStore.FindByGoogleID(googleID)
	if err == nil {
		// User found, return it
		return user, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// A different error occurred during lookup
		return nil, fmt.Errorf("error checking Google user: %w", err)
	}

	// User not found, create a new one
	newUser := &models.User{
		Email:    email,
		Name:     name,
		GoogleID: googleID,
	}

	err = s.userStore.Create(newUser)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return newUser, nil
}

// GetUserByID retrieves a user by their ID.
func (s *UserService) GetUserByID(userID uint) (*models.User, error) {
	user, err := s.userStore.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("error finding user: %w", err)
	}
	return user, nil
}
