package services

import (
	"fmt"

	"linkshrink/internal/core/models"
	"linkshrink/internal/store"
)

// PurchaseService provides business logic for managing purchases.
type PurchaseService struct {
	purchaseStore *store.PurchaseStore
}

// NewPurchaseService creates a new PurchaseService.
func NewPurchaseService(purchaseStore *store.PurchaseStore) *PurchaseService {
	return &PurchaseService{
		purchaseStore: purchaseStore,
	}
}

// SavePurchase creates a record of a completed purchase transaction
func (s *PurchaseService) SavePurchase(shortCode, targetURL, method string, price int64, paidRouteID uint, paymentPayload, settleResponse string) error {
	// Create and save the purchase record
	purchase := &models.Purchase{
		ShortCode:      shortCode,
		TargetURL:      targetURL,
		Method:         method,
		Price:          price,
		PaymentPayload: paymentPayload,
		SettleResponse: settleResponse,
		PaidRouteID:    paidRouteID,
	}

	if err := s.purchaseStore.Create(purchase); err != nil {
		return fmt.Errorf("failed to record purchase: %w", err)
	}

	return nil
}

// ListPurchasesByShortCode retrieves all purchases for a specific shortcode.
func (s *PurchaseService) ListPurchasesByShortCode(shortCode string) ([]models.Purchase, error) {
	return s.purchaseStore.ListByShortCode(shortCode)
}

// ListPurchasesByUserID retrieves all purchases for a specific user ID.
func (s *PurchaseService) ListPurchasesByUserID(userID uint) ([]models.Purchase, error) {
	return s.purchaseStore.ListByUserID(userID)
}

// GetDashboardStats retrieves daily purchase stats for the dashboard
func (s *PurchaseService) GetDashboardStats(userID uint, days int) ([]store.DailyStats, int64, int, error) {
	return s.purchaseStore.GetDailyStatsByUserID(userID, days)
}
