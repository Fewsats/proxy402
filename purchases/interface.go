package purchases

import (
	"context"
	"errors"
)

// Custom errors for purchase operations
var (
	ErrPurchaseNotFound = errors.New("purchase not found")
	ErrNoStats          = errors.New("no purchase statistics available")
)

// Store provides access to the purchase storage.
type Store interface {
	// Create inserts a new purchase in the database and returns the ID.
	CreatePurchase(ctx context.Context, purchase *Purchase) (uint64, error)

	// ListPurchasesByUserID retrieves all purchases for a specific user.
	ListPurchasesByUserID(ctx context.Context, userID uint) ([]Purchase, error)

	// ListPurchasesByShortCode retrieves all purchases for a specific shortcode.
	ListPurchasesByShortCode(ctx context.Context, shortCode string) ([]Purchase, error)

	// GetDailyStatsByUserID retrieves daily purchase stats for a user.
	GetDailyStatsByUserID(ctx context.Context, userID uint, days int) ([]DailyStats, int64, int, error)
}
