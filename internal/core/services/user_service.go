package services

import (
	"crypto/rand"
	"encoding/hex"
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

	// Generate a unique secret for the new user
	secretBytes := make([]byte, 16) // 16 bytes = 32 hex characters
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, fmt.Errorf("failed to generate proxy secret: %w", err)
	}
	proxySecret := hex.EncodeToString(secretBytes)

	newUser := &models.User{
		Email:          email,
		Name:           name,
		GoogleID:       googleID,
		Proxy402Secret: proxySecret, // Set the generated secret
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
