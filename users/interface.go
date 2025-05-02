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
}
