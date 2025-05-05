package users

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
)

// UserService provides user-related business logic.
type UserService struct {
	logger *slog.Logger
	store  Store
}

// NewUserService creates a new UserService.
func NewUserService(logger *slog.Logger, store Store) *UserService {
	return &UserService{logger: logger, store: store}
}

// FindOrCreateUser finds a user by Google ID or creates a new one.
func (s *UserService) FindOrCreateUser(ctx context.Context, email, name,
	googleID string) (*User, error) {

	// Try to find user by Google ID
	user, err := s.store.FindUserByGoogleID(ctx, googleID)
	if err == nil {
		// User found, return it
		return user, nil
	} else if !errors.Is(err, ErrUserNotFound) {
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

	newUser := &User{
		Email:          email,
		Name:           name,
		GoogleID:       googleID,
		Proxy402Secret: proxySecret,
	}

	id, err := s.store.CreateUser(ctx, newUser)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	newUser.ID = uint64(id)

	return newUser, nil
}

// GetUserByID retrieves a user by their ID.
func (s *UserService) GetUserByID(ctx context.Context, userID uint64) (*User, error) {
	user, err := s.store.FindUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("error finding user: %w", err)
	}
	return user, nil
}

// UpdateProxySecret generates and updates the user's proxy secret.
func (s *UserService) UpdateProxySecret(ctx context.Context, userID uint64) (*User, error) {
	// Generate a new secret
	secretBytes := make([]byte, 16) // 16 bytes = 32 hex characters
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, fmt.Errorf("failed to generate proxy secret: %w", err)
	}
	newSecret := hex.EncodeToString(secretBytes)

	// Update the user's secret in the database
	user, err := s.store.UpdateUserProxySecret(ctx, userID, newSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to update proxy secret: %w", err)
	}

	return user, nil
}

// UpdatePaymentAddress validates and updates the user's payment address.
func (s *UserService) UpdatePaymentAddress(ctx context.Context, userID uint64,
	address string) (*User, error) {

	// Validate payment address
	if address != "" {
		// Check that it starts with 0x
		if !regexp.MustCompile(`^0x[a-fA-F0-9]{40}$`).MatchString(address) {
			return nil, fmt.Errorf("invalid payment address format, must start with 0x followed by 40 hex characters")
		}
	}

	// Update the user's payment address
	user, err := s.store.UpdateUserPaymentAddress(ctx, userID, address)
	if err != nil {
		return nil, fmt.Errorf("failed to update payment address: %w", err)
	}

	return user, nil
}
