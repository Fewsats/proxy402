package models

import (
	"gorm.io/gorm"
)

// User represents a registered user in the system.
type User struct {
	gorm.Model        // Includes fields like ID, CreatedAt, UpdatedAt, DeletedAt
	Username   string `gorm:"uniqueIndex;not null" json:"username"`
	Password   string `gorm:"not null" json:"-"` // Store hashed password, exclude from JSON
	// Links      []Link `gorm:"foreignKey:UserID" json:"-"` // User has many Links - REMOVED
	PaidRoutes []PaidRoute `gorm:"foreignKey:UserID" json:"-"` // User has many PaidRoutes
}

// Link represents a shortened URL. - REMOVED
/*
type Link struct {
	gorm.Model             // Includes fields like ID, CreatedAt, UpdatedAt, DeletedAt
	OriginalURL string     `gorm:"not null" json:"original_url"`
	ShortCode   string     `gorm:"uniqueIndex;not null" json:"short_code"`
	UserID      uint       `gorm:"not null" json:"-"`                                       // Foreign key to User
	User        User       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"` // Belongs to User
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`                                    // Optional expiration time
	VisitCount  int64      `gorm:"default:0" json:"visit_count"`
}
*/

// PaidRoute represents a configurable, paid API route proxied by the service.
type PaidRoute struct {
	gorm.Model
	ShortCode string `gorm:"uniqueIndex;not null" json:"short_code"`
	TargetURL string `gorm:"not null" json:"target_url"`
	Method    string `gorm:"not null" json:"method"` // GET, POST, PUT, DELETE, PATCH
	// Store price as string for precision, map to NUMERIC in DB.
	// Assumed to be the crypto amount (e.g., ETH) required by facilitator.
	Price        string `gorm:"type:numeric;not null" json:"price"`
	UserID       uint   `gorm:"not null" json:"-"` // User who owns/created this route
	User         User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	IsEnabled    bool   `gorm:"default:true" json:"is_enabled"`
	PaymentCount int64  `gorm:"default:0" json:"payment_count"` // Track successful payments/accesses
}
