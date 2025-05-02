package routes

import (
	"context"
	"errors"

	"linkshrink/internal/core/models"
)

// Custom errors for route operations
var (
	ErrRouteNotFound     = errors.New("route not found")
	ErrRouteDisabled     = errors.New("route is disabled")
	ErrRouteNoPermission = errors.New("you do not have permission to access this route")
)

// Store provides access to the paid route storage.
type Store interface {
	// CreateRoute inserts a new paid route in the database and returns the route.
	CreateRoute(ctx context.Context, route *models.PaidRoute) (*models.PaidRoute, error)

	// FindRouteByID retrieves a paid route by ID.
	FindRouteByID(ctx context.Context, id uint) (*models.PaidRoute, error)

	// FindRouteByShortCode retrieves a paid route by short code.
	FindRouteByShortCode(ctx context.Context, shortCode string) (*models.PaidRoute, error)

	// FindEnabledRouteByShortCode retrieves an enabled paid route by short code.
	FindEnabledRouteByShortCode(ctx context.Context, shortCode string) (*models.PaidRoute, error)

	// ListUserRoutes retrieves all paid routes for a specific user.
	ListUserRoutes(ctx context.Context, userID uint) ([]models.PaidRoute, error)

	// DeleteRoute soft-deletes a paid route.
	DeleteRoute(ctx context.Context, routeID uint, userID uint) error

	// IncrementRouteAttemptCount increments the attempt_count for a route.
	IncrementRouteAttemptCount(ctx context.Context, shortCode string) error

	// IncrementRoutePaymentCount increments the payment_count for a route.
	IncrementRoutePaymentCount(ctx context.Context, shortCode string) error

	// IncrementRouteAccessCount increments the access_count for a route.
	IncrementRouteAccessCount(ctx context.Context, shortCode string) error
}
