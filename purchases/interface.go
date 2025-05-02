package purchases

import (
	"context"
	"errors"

	"linkshrink/internal/core/models"
)

// Custom errors for purchase operations
var (
	ErrPurchaseNotFound = errors.New("purchase not found")
	ErrNoStats          = errors.New("no purchase statistics available")
)


// Store provides access to the purchase storage.
type Store interface {
	// Create inserts a new purchase in the database and returns the ID.
	CreatePurchase(ctx context.Context, purchase *models.Purchase) (uint64, error)

	// ListPurchasesByUserID retrieves all purchases for a specific user.
	ListPurchasesByUserID(ctx context.Context, userID uint) ([]models.Purchase, error)

	// ListPurchasesByShortCode retrieves all purchases for a specific shortcode.
	ListPurchasesByShortCode(ctx context.Context, shortCode string) ([]models.Purchase, error)

	// GetDailyStatsByUserID retrieves daily purchase stats for a user.
	GetDailyStatsByUserID(ctx context.Context, userID uint, days int) ([]DailyStats, int64, int, error)
}
