package models

import (
	"gorm.io/gorm"
)

// User represents a registered user in the system.
type User struct {
	gorm.Model
	Email      string      `gorm:"uniqueIndex;not null" json:"email"` // Email is the primary identifier
	Name       string      `json:"name,omitempty"`                    // User's name from Google
	GoogleID   string      `gorm:"index;not null" json:"-"`           // Google user ID
	PaidRoutes []PaidRoute `gorm:"foreignKey:UserID" json:"-"`        // User has many PaidRoutes
}

// PaidRoute represents a configurable, paid API route proxied by the service.
type PaidRoute struct {
	gorm.Model
	ShortCode string `gorm:"uniqueIndex;not null" json:"short_code"`
	TargetURL string `gorm:"not null" json:"target_url"`
	Method    string `gorm:"not null" json:"method"` // GET, POST, PUT, DELETE, PATCH
	// Store price as string for precision, map to NUMERIC in DB.
	// Assumed to be the crypto amount (e.g., ETH) required by facilitator.
	// TODO change into base units which is 10^6
	Price        string `gorm:"type:numeric;not null" json:"price"`
	UserID       uint   `gorm:"not null" json:"-"` // User who owns/created this route
	User         User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	IsEnabled    bool   `gorm:"default:true" json:"is_enabled"`
	PaymentCount int64  `gorm:"default:0" json:"payment_count"` // Track successful payments/accesses
}
