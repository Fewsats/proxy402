package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"linkshrink/store/sqlc"
	"linkshrink/users"
)

// Create inserts a new user in the database.
func (s *Store) CreateUser(ctx context.Context, user *users.User) (uint64, error) {
	userID, err := s.queries.CreateUser(ctx, sqlc.CreateUserParams{
		Email:          user.Email,
		Name:           user.Name,
		GoogleID:       user.GoogleID,
		Proxy402Secret: user.Proxy402Secret,
		PaymentAddress: "",
		CreatedAt:      s.clock.Now(),
		UpdatedAt:      s.clock.Now(),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	return uint64(userID), nil
}

// FindByID retrieves a user by ID.
func (s *Store) FindUserByID(ctx context.Context, id uint) (*users.User, error) {
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
func (s *Store) FindUserByGoogleID(ctx context.Context, googleID string) (*users.User, error) {
	dbUser, err := s.queries.GetUserByGoogleID(ctx, googleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, users.ErrUserNotFound
		}
		return nil, err
	}

	return convertToUserModel(dbUser), nil
}

// Helper function to convert sqlc User to users.User
func convertToUserModel(dbUser sqlc.User) *users.User {
	return &users.User{
		ID:             uint(dbUser.ID),
		Email:          dbUser.Email,
		Name:           dbUser.Name,
		GoogleID:       dbUser.GoogleID,
		Proxy402Secret: dbUser.Proxy402Secret,
		PaymentAddress: dbUser.PaymentAddress,
		CreatedAt:      dbUser.CreatedAt,
		UpdatedAt:      dbUser.UpdatedAt,
	}
}

// UpdateUserProxySecret updates a user's proxy secret.
func (s *Store) UpdateUserProxySecret(ctx context.Context, id uint, secret string) error {
	err := s.queries.UpdateUserProxySecret(ctx, sqlc.UpdateUserProxySecretParams{
		ID:             int64(id),
		Proxy402Secret: secret,
		UpdatedAt:      s.clock.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to update user proxy secret: %w", err)
	}
	return nil
}

// UpdateUserPaymentAddress updates a user's payment address.
func (s *Store) UpdateUserPaymentAddress(ctx context.Context, id uint, address string) error {
	err := s.queries.UpdateUserPaymentAddress(ctx, sqlc.UpdateUserPaymentAddressParams{
		ID:             int64(id),
		PaymentAddress: address,
		UpdatedAt:      s.clock.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to update user payment address: %w", err)
	}
	return nil
}
