package services

import (
	"context"
	"log/slog"

	"linkshrink/internal/core/models"
	"linkshrink/purchases"
)

// PurchaseService provides business logic for managing purchases.
type PurchaseService struct {
	logger *slog.Logger
	store  purchases.Store
}

// NewPurchaseService creates a new PurchaseService.
func NewPurchaseService(logger *slog.Logger, store purchases.Store) *PurchaseService {
	return &PurchaseService{
		logger: logger,
		store:  store,
	}
}

// ListPurchasesByUserID retrieves all purchases for a specific user ID.
func (s *PurchaseService) ListPurchasesByUserID(ctx context.Context, userID uint) ([]models.Purchase, error) {
	return s.store.ListPurchasesByUserID(ctx, userID)
}

// GetDashboardStats retrieves daily purchase stats for the dashboard
func (s *PurchaseService) GetDashboardStats(ctx context.Context, userID uint, days int) ([]purchases.DailyStats, int64, int, error) {
	return s.store.GetDailyStatsByUserID(ctx, userID, days)
}

// Create creates a new purchase.
func (s *PurchaseService) CreatePurchase(ctx context.Context, purchase *models.Purchase) (uint64, error) {
	return s.store.CreatePurchase(ctx, purchase)
}
