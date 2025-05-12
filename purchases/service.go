package purchases

import (
	"context"
	"log/slog"
)

// PurchaseService provides business logic for managing purchases.
type PurchaseService struct {
	logger *slog.Logger
	store  Store
}

// NewPurchaseService creates a new PurchaseService.
func NewPurchaseService(logger *slog.Logger, store Store) *PurchaseService {
	return &PurchaseService{
		logger: logger,
		store:  store,
	}
}

// ListPurchasesByUserID retrieves all purchases for a specific user ID.
func (s *PurchaseService) ListPurchasesByUserID(ctx context.Context, userID uint64) ([]Purchase, error) {
	return s.store.ListPurchasesByUserID(ctx, userID)
}

// GetDashboardStats retrieves daily purchase stats for the dashboard
func (s *PurchaseService) GetDashboardStats(ctx context.Context, userID uint64, days uint64) ([]DailyStats, error) {
	return s.store.GetDailyStatsByUserID(ctx, userID, days)
}

// Create creates a new purchase.
func (s *PurchaseService) CreatePurchase(ctx context.Context, purchase *Purchase) (uint64, error) {
	return s.store.CreatePurchase(ctx, purchase)
}

// GetPurchaseByRouteIDAndPaymentHeader retrieves a purchase if it exists for the given route and payment header.
func (s *PurchaseService) GetPurchaseByRouteIDAndPaymentHeader(ctx context.Context, routeID uint64, paymentHeader string) (*Purchase, error) {
	if paymentHeader == "" { // Basic validation
		return nil, ErrPaymentHeader
	}

	return s.store.GetPurchaseByRouteIDAndPaymentHeader(ctx, routeID, paymentHeader)
}

// IncrementCreditsUsed calls the store to increment the used credits for a purchase.
func (s *PurchaseService) IncrementCreditsUsed(ctx context.Context, purchaseID uint64) error {
	return s.store.IncrementPurchaseCreditsUsed(ctx, purchaseID)
}
