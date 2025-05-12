package purchases

import (
	"context"
	"errors"
)

// Custom errors for purchase operations
var (
	ErrPurchaseNotFound = errors.New("purchase not found")
	ErrNoStats          = errors.New("no purchase statistics available")
	ErrPaymentHeader    = errors.New("payment header cannot be empty")
)

// Store provides access to the purchase storage.
type Store interface {
	// Create inserts a new purchase in the database and returns the ID.
	CreatePurchase(ctx context.Context, purchase *Purchase) (uint64, error)

	// ListPurchasesByUserID retrieves all purchases for a specific user.
	ListPurchasesByUserID(ctx context.Context, userID uint64) ([]Purchase, error)

	// ListPurchasesByShortCode retrieves all purchases for a specific shortcode.
	ListPurchasesByShortCode(ctx context.Context,
		shortCode string) ([]Purchase, error)

	// GetDailyStatsByUserID retrieves daily purchase stats for a user.
	GetDailyStatsByUserID(ctx context.Context, userID uint64,
		days uint64) ([]DailyStats, error)

	// GetPurchaseByRouteIDAndPaymentHeader retrieves a purchase if it exists for the given route and payment header.
	GetPurchaseByRouteIDAndPaymentHeader(ctx context.Context, routeID uint64, paymentHeader string) (*Purchase, error)

	// IncrementPurchaseCreditsUsed increments the credits used for a purchase.
	IncrementPurchaseCreditsUsed(ctx context.Context, purchaseID uint64) error
}
