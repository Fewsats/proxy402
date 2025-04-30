package store

import (
	"linkshrink/internal/core/models"

	"gorm.io/gorm"
)

// PurchaseStore defines methods for interacting with purchase data.
type PurchaseStore struct {
	db *gorm.DB
}

// NewPurchaseStore creates a new PurchaseStore.
func NewPurchaseStore(db *gorm.DB) *PurchaseStore {
	return &PurchaseStore{db: db}
}

// Create inserts a new purchase record into the database.
func (s *PurchaseStore) Create(purchase *models.Purchase) error {
	result := s.db.Create(purchase)
	return result.Error
}

// ListByShortCode retrieves all purchases for a specific short code.
func (s *PurchaseStore) ListByShortCode(shortCode string) ([]models.Purchase, error) {
	var purchases []models.Purchase
	result := s.db.Where("short_code = ?", shortCode).Order("created_at desc").Find(&purchases)
	return purchases, result.Error
}

// ListByUserID retrieves all purchases for a specific user.
// This joins with the PaidRoute table to filter by UserID.
func (s *PurchaseStore) ListByUserID(userID uint) ([]models.Purchase, error) {
	var purchases []models.Purchase
	result := s.db.Joins("JOIN paid_routes ON purchases.paid_route_id = paid_routes.id").
		Where("paid_routes.user_id = ?", userID).
		Order("purchases.created_at desc").
		Find(&purchases)
	return purchases, result.Error
}
