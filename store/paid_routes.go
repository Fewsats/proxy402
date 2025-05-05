package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"linkshrink/routes"
	"linkshrink/store/sqlc"
	"linkshrink/utils"
)

const (
	// DefaultLength is the standard length for generated short codes.
	DefaultLength = 7
	// charset is the set of characters to use for generating short codes.
	// Using base64 URL safe characters.
	charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"
)

// CreateRoute inserts a new paid route in the database and returns the ID.
func (s *Store) CreateRoute(ctx context.Context, route *routes.PaidRoute) (*routes.PaidRoute, error) {

	shortCode, err := utils.GenerateSecureShortCode(10)
	if err != nil {
		return nil, fmt.Errorf("failed to generate unique short code: %w", err)
	}

	now := s.clock.Now()

	params := sqlc.CreatePaidRouteParams{
		ShortCode: shortCode,
		TargetUrl: route.TargetURL,
		Method:    route.Method,
		Price:     int32(route.Price),
		IsTest:    route.IsTest,
		UserID:    int64(route.UserID),
		IsEnabled: route.IsEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}

	dbRoute, err := s.queries.CreatePaidRoute(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create paid route: %w", err)
	}

	return convertToPaidRouteModel(dbRoute), nil
}

// FindRouteByID retrieves a paid route by ID.
func (s *Store) FindRouteByID(ctx context.Context, id uint64) (*routes.PaidRoute, error) {
	dbRoute, err := s.queries.GetPaidRouteByID(ctx, int64(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, routes.ErrRouteNotFound
		}
		return nil, err
	}

	return convertToPaidRouteModel(dbRoute), nil
}

// FindRouteByShortCode retrieves a paid route by short code.
func (s *Store) FindRouteByShortCode(ctx context.Context, shortCode string) (*routes.PaidRoute, error) {
	dbRoute, err := s.queries.GetPaidRouteByShortCode(ctx, shortCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, routes.ErrRouteNotFound
		}
		return nil, err
	}

	return convertToPaidRouteModel(dbRoute), nil
}

// FindEnabledRouteByShortCode retrieves an enabled paid route by short code.
func (s *Store) FindEnabledRouteByShortCode(ctx context.Context, shortCode string) (*routes.PaidRoute, error) {
	dbRoute, err := s.queries.GetEnabledPaidRouteByShortCode(ctx, shortCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, routes.ErrRouteNotFound
		}
		return nil, err
	}

	return convertToPaidRouteModel(dbRoute), nil
}

// ListUserRoutes retrieves all paid routes for a specific user.
func (s *Store) ListUserRoutes(ctx context.Context, userID uint64) ([]routes.PaidRoute, error) {
	dbRoutes, err := s.queries.ListUserPaidRoutes(ctx, int64(userID))
	if err != nil {
		return nil, err
	}

	routes := make([]routes.PaidRoute, len(dbRoutes))
	for i, dbRoute := range dbRoutes {
		routes[i] = *convertToPaidRouteModel(dbRoute)
	}

	return routes, nil
}

// DeleteRoute soft-deletes a paid route.
func (s *Store) DeleteRoute(ctx context.Context, routeID uint64, userID uint64) error {
	err := s.queries.DeletePaidRoute(ctx, sqlc.DeletePaidRouteParams{
		ID:     int64(routeID),
		UserID: int64(userID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return routes.ErrRouteNoPermission
		}
		return err
	}
	return nil
}

// IncrementRouteAttemptCount increments the attempt_count for a route.
func (s *Store) IncrementRouteAttemptCount(ctx context.Context, shortCode string) error {
	err := s.queries.IncrementAttemptCount(ctx, sqlc.IncrementAttemptCountParams{
		ShortCode: shortCode,
	})
	if err != nil {
		return fmt.Errorf("failed to increment attempt count: %w", err)
	}
	return nil
}

// IncrementRoutePaymentCount increments the payment_count for a route.
func (s *Store) IncrementRoutePaymentCount(ctx context.Context, shortCode string) error {
	err := s.queries.IncrementPaymentCount(ctx, sqlc.IncrementPaymentCountParams{
		ShortCode: shortCode,
	})
	if err != nil {
		return fmt.Errorf("failed to increment payment count: %w", err)
	}
	return nil
}

// IncrementRouteAccessCount increments the access_count for a route.
func (s *Store) IncrementRouteAccessCount(ctx context.Context, shortCode string) error {
	err := s.queries.IncrementAccessCount(ctx, sqlc.IncrementAccessCountParams{
		ShortCode: shortCode,
	})
	if err != nil {
		return fmt.Errorf("failed to increment access count: %w", err)
	}
	return nil
}

// CheckShortCodeExists checks if a short code already exists.
func (s *Store) CheckShortCodeExists(ctx context.Context, shortCode string) (bool, error) {
	exists, err := s.queries.CheckShortCodeExists(ctx, shortCode)
	if err != nil {
		return false, fmt.Errorf("failed to check if short code exists: %w", err)
	}
	return exists, nil
}

// GenerateUniqueShortCode generates a unique short code.
// It will retry up to maxRetries times if codes are already taken.
func (s *Store) GenerateUniqueShortCode(ctx context.Context,
	length int, maxRetries int) (string, error) {

	if length <= 0 {
		length = DefaultLength // Use the default from the shortener package
	}

	if maxRetries <= 0 {
		maxRetries = 10 // Default max retries
	}

	for i := 0; i < maxRetries; i++ {
		shortCode, err := utils.GenerateSecureShortCode(length)
		if err != nil {
			return "", fmt.Errorf("failed to generate short code: %w", err)
		}

		exists, err := s.CheckShortCodeExists(ctx, shortCode)
		if err != nil {
			return "", err
		}

		if !exists {
			return shortCode, nil
		}
	}

	return "", errors.New("failed to generate a unique short code after multiple attempts")
}

// Helper function to convert sqlc PaidRoute to models.PaidRoute
func convertToPaidRouteModel(dbRoute sqlc.PaidRoute) *routes.PaidRoute {
	route := &routes.PaidRoute{
		ID:           uint64(dbRoute.ID),
		ShortCode:    dbRoute.ShortCode,
		TargetURL:    dbRoute.TargetUrl,
		Method:       dbRoute.Method,
		Price:        uint64(dbRoute.Price),
		IsTest:       dbRoute.IsTest,
		UserID:       uint64(dbRoute.UserID),
		IsEnabled:    dbRoute.IsEnabled,
		AttemptCount: uint64(dbRoute.AttemptCount),
		PaymentCount: uint64(dbRoute.PaymentCount),
		AccessCount:  uint64(dbRoute.AccessCount),
		CreatedAt:    dbRoute.CreatedAt,
		UpdatedAt:    dbRoute.UpdatedAt,
	}

	// Convert DeletedAt if it exists
	if dbRoute.DeletedAt.Valid {
		deletedAt := dbRoute.DeletedAt.Time
		route.DeletedAt = &deletedAt
	}

	return route
}
