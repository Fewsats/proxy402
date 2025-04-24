package store

import (
	"errors"

	"gorm.io/gorm"

	"linkshrink/internal/core/models"
)

// LinkStore defines methods for interacting with link data.
type LinkStore struct {
	db *gorm.DB
}

// NewLinkStore creates a new LinkStore.
func NewLinkStore(db *gorm.DB) *LinkStore {
	return &LinkStore{db: db}
}

// Create inserts a new link into the database.
func (s *LinkStore) Create(link *models.Link) error {
	result := s.db.Create(link)
	return result.Error
}

// FindByShortCode retrieves a link by its short code.
// Returns gorm.ErrRecordNotFound if the link doesn't exist.
func (s *LinkStore) FindByShortCode(shortCode string) (*models.Link, error) {
	var link models.Link
	result := s.db.Where("short_code = ?", shortCode).First(&link)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, result.Error
	}
	return &link, nil
}

// IncrementVisitCount increases the visit count for a given short code.
func (s *LinkStore) IncrementVisitCount(shortCode string) error {
	result := s.db.Model(&models.Link{}).Where("short_code = ?", shortCode).UpdateColumn("visit_count", gorm.Expr("visit_count + ?", 1))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// This could happen if the record was deleted between the Find and Update,
		// or if the shortCode doesn't exist. The Find should catch the latter.
		return gorm.ErrRecordNotFound // Or a custom error
	}
	return nil
}

// FindByUserID retrieves all links created by a specific user.
func (s *LinkStore) FindByUserID(userID uint) ([]models.Link, error) {
	var links []models.Link
	result := s.db.Where("user_id = ?", userID).Find(&links)
	return links, result.Error
}

// Delete removes a link from the database.
func (s *LinkStore) Delete(id uint, userID uint) error {
	// Ensure the user owns the link they are trying to delete
	result := s.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Link{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// Either the link doesn't exist, or the user doesn't own it
		return gorm.ErrRecordNotFound // Or a more specific permission error
	}
	return nil
}
