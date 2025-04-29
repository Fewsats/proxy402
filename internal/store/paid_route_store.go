package store

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"linkshrink/internal/core/models"
)

// ErrRouteNotFound is returned when a route is not found.
var ErrRouteNotFound = errors.New("route not found")

// PaidRouteStore defines methods for interacting with paid route data.
type PaidRouteStore struct {
	db *gorm.DB
}

// NewPaidRouteStore creates a new PaidRouteStore.
func NewPaidRouteStore(db *gorm.DB) *PaidRouteStore {
	return &PaidRouteStore{db: db}
}

// Create inserts a new paid route into the database.
func (s *PaidRouteStore) Create(route *models.PaidRoute) error {
	result := s.db.Create(route)
	return result.Error
}

// FindByShortCode retrieves an enabled paid route by its short code.
// Returns gorm.ErrRecordNotFound if the route doesn't exist or is not enabled.
func (s *PaidRouteStore) FindByShortCode(shortCode string) (*models.PaidRoute, error) {
	var route models.PaidRoute
	result := s.db.Where("short_code = ? AND is_enabled = ?", shortCode, true).First(&route)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, result.Error
	}
	return &route, nil
}

// CheckShortCodeExists checks if a short code already exists in the database (enabled or not).
func (s *PaidRouteStore) CheckShortCodeExists(shortCode string) (bool, error) {
	var count int64
	result := s.db.Model(&models.PaidRoute{}).Where("short_code = ?", shortCode).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// IncrementPaymentCount increases the payment count for a given short code.
func (s *PaidRouteStore) IncrementPaymentCount(shortCode string) error {
	// Only increment if the route is enabled and found
	result := s.db.Model(&models.PaidRoute{}).
		Where("short_code = ? AND is_enabled = ?", shortCode, true).
		UpdateColumn("payment_count", gorm.Expr("payment_count + ?", 1))

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// Could be not found, or already disabled between check and update.
		return gorm.ErrRecordNotFound // Indicate route wasn't found/updated
	}
	return nil
}

// ListByUserID retrieves all paid routes created by a specific user.
func (s *PaidRouteStore) ListByUserID(userID uint) ([]models.PaidRoute, error) {
	var routes []models.PaidRoute
	// Order by creation time, newest first
	result := s.db.Where("user_id = ?", userID).Order("created_at desc").Find(&routes)
	return routes, result.Error
}

// Delete removes a paid route from the database, ensuring user ownership.
func (s *PaidRouteStore) Delete(routeID uint, userID uint) error {
	// Ensure the user owns the route they are trying to delete
	result := s.db.Where("id = ? AND user_id = ?", routeID, userID).Delete(&models.PaidRoute{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// Either the route doesn't exist, or the user doesn't own it
		return gorm.ErrRecordNotFound // Or a more specific permission error
	}
	return nil
}

// IncrementAttemptCount increments the attempt_count for a route identified by shortCode.
func (s *PaidRouteStore) IncrementAttemptCount(shortCode string) error {
	result := s.db.Model(&models.PaidRoute{}).
		Where("short_code = ?", shortCode).
		UpdateColumn("attempt_count", gorm.Expr("attempt_count + 1"))

	if result.Error != nil {
		return fmt.Errorf("db error incrementing attempt count for %s: %w", shortCode, result.Error)
	}
	if result.RowsAffected == 0 {
		// Use the defined error
		return fmt.Errorf("route %s not found or no change needed for attempt count: %w", shortCode, ErrRouteNotFound)
	}
	return nil
}

// IncrementAccessCount increments the access_count for a route identified by shortCode.
func (s *PaidRouteStore) IncrementAccessCount(shortCode string) error {
	result := s.db.Model(&models.PaidRoute{}).
		Where("short_code = ?", shortCode).
		UpdateColumn("access_count", gorm.Expr("access_count + 1"))

	if result.Error != nil {
		return fmt.Errorf("db error incrementing access count for %s: %w", shortCode, result.Error)
	}
	if result.RowsAffected == 0 {
		// Use the defined error
		return fmt.Errorf("route %s not found or no change needed for access count: %w", shortCode, ErrRouteNotFound)
	}
	return nil
}

// TODO: Add methods for Update etc. as needed later.
