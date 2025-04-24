package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a registered user in the system.
type User struct {
	gorm.Model        // Includes fields like ID, CreatedAt, UpdatedAt, DeletedAt
	Username   string `gorm:"uniqueIndex;not null" json:"username"`
	Password   string `gorm:"not null" json:"-"`          // Store hashed password, exclude from JSON
	Links      []Link `gorm:"foreignKey:UserID" json:"-"` // User has many Links
}

// Link represents a shortened URL.
type Link struct {
	gorm.Model             // Includes fields like ID, CreatedAt, UpdatedAt, DeletedAt
	OriginalURL string     `gorm:"not null" json:"original_url"`
	ShortCode   string     `gorm:"uniqueIndex;not null" json:"short_code"`
	UserID      uint       `gorm:"not null" json:"-"`                                       // Foreign key to User
	User        User       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"` // Belongs to User
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`                                    // Optional expiration time
	VisitCount  int64      `gorm:"default:0" json:"visit_count"`
}
