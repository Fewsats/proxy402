package services

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"linkshrink/internal/auth"
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

// Register creates a new user, hashes their password, and saves them.
func (s *UserService) Register(username, password string) (*models.User, error) {
	// Check if username already exists
	_, err := s.userStore.FindByUsername(username)
	if err == nil {
		// User found, username is taken
		return nil, errors.New("username already taken")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// A different error occurred during lookup
		return nil, fmt.Errorf("error checking username: %w", err)
	}
	// User not found, proceed with registration

	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Username: username,
		Password: hashedPassword,
	}

	if err := s.userStore.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Login authenticates a user and generates a JWT if successful.
func (s *UserService) Login(username, password string) (string, error) {
	user, err := s.userStore.FindByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("invalid username or password")
		}
		return "", fmt.Errorf("error finding user: %w", err)
	}

	if !auth.CheckPasswordHash(password, user.Password) {
		return "", errors.New("invalid username or password")
	}

	token, err := auth.GenerateJWT(user.ID, user.Username)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return token, nil
}
