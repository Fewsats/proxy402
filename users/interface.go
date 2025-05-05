package users

import (
	"context"
	"errors"
)

// Custom errors for user operations
var (
	ErrUserNotFound = errors.New("user not found")
)

// Store provides access to the user storage.
type Store interface {
	// CreateUser inserts a new user in the database.
	CreateUser(ctx context.Context, user *User) (uint64, error)

	// FindUserByID retrieves a user by ID.
	FindUserByID(ctx context.Context, id uint) (*User, error)

	// FindUserByGoogleID retrieves a user by Google ID.
	FindUserByGoogleID(ctx context.Context, googleID string) (*User, error)

	// UpdateUserProxySecret updates a user's proxy secret.
	UpdateUserProxySecret(ctx context.Context, id uint, secret string) error

	// UpdateUserPaymentAddress updates a user's payment address.
	UpdateUserPaymentAddress(ctx context.Context, id uint, address string) error
}
