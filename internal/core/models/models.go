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
	// Store price as int64 representing base units (USDC * 10^6)
	Price        int64 `gorm:"not null" json:"price"`
	IsTest       bool  `gorm:"not null" json:"is_test"`
	UserID       uint  `gorm:"not null" json:"-"` // User who owns/created this route
	User         User  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	IsEnabled    bool  `gorm:"default:true" json:"is_enabled"`
	AttemptCount int64 `gorm:"default:0" json:"attempt_count"` // Track payment attempts (no payment header provided)
	PaymentCount int64 `gorm:"default:0" json:"payment_count"` // Track successful payments (payment for x402)
	AccessCount  int64 `gorm:"default:0" json:"access_count"`  // Track successful accesses (payment header provided)
}

// Purchase records information about a successful payment transaction
type Purchase struct {
	gorm.Model
	ShortCode      string    `gorm:"index;not null" json:"short_code"`
	TargetURL      string    `gorm:"not null" json:"target_url"`
	Method         string    `gorm:"not null" json:"method"`
	Price          int64     `gorm:"not null" json:"price"`
	PaymentPayload string    `gorm:"type:jsonb;not null" json:"payment_payload"` // X-Payment header as JSON
	SettleResponse string    `gorm:"type:jsonb;not null" json:"settle_response"` // Settled response as base64
	PaidRouteID    uint      `gorm:"index;not null" json:"-"`                    // Associated PaidRoute
	PaidRoute      PaidRoute `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
}
