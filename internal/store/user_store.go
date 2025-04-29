package store

import (
	"errors"

	"gorm.io/gorm"

	"linkshrink/internal/core/models"
)

// UserStore defines methods for interacting with user data.
type UserStore struct {
	db *gorm.DB
}

// NewUserStore creates a new UserStore.
func NewUserStore(db *gorm.DB) *UserStore {
	return &UserStore{db: db}
}

// Create inserts a new user into the database.
func (s *UserStore) Create(user *models.User) error {
	result := s.db.Create(user)
	return result.Error
}

// FindByID retrieves a user by their ID.
// Returns gorm.ErrRecordNotFound if the user doesn't exist.
func (s *UserStore) FindByID(id uint) (*models.User, error) {
	var user models.User
	result := s.db.First(&user, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, result.Error
	}
	return &user, nil
}

// FindByGoogleID retrieves a user by their Google ID.
// Returns gorm.ErrRecordNotFound if the user doesn't exist.
func (s *UserStore) FindByGoogleID(googleID string) (*models.User, error) {
	var user models.User
	result := s.db.Where("google_id = ?", googleID).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, result.Error
	}
	return &user, nil
}

// Update updates a user in the database.
func (s *UserStore) Update(user *models.User) error {
	result := s.db.Save(user)
	return result.Error
}
