package models

import (
	"time"
)

// User represents a registered user in the system.
type User struct {
	ID         uint        `json:"-"`
	Email      string      `json:"email"`          // Email is the primary identifier
	Name       string      `json:"name,omitempty"` // User's name from Google
	GoogleID   string      `json:"-"`              // Google user ID
	PaidRoutes []PaidRoute `json:"-"`              // User has many PaidRoutes
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// PaidRoute represents a configurable, paid API route proxied by the service.
type PaidRoute struct {
	ID        uint   `json:"id"`
	ShortCode string `json:"short_code"`
	TargetURL string `json:"target_url"`
	Method    string `json:"method"` // GET, POST, PUT, DELETE, PATCH
	// Store price as int64 representing base units (USDC * 10^6)
	Price        int64      `json:"price"`
	IsTest       bool       `json:"is_test"`
	UserID       uint       `json:"-"` // User who owns/created this route
	User         User       `json:"-"`
	IsEnabled    bool       `json:"is_enabled"`
	AttemptCount int64      `json:"attempt_count"` // Track payment attempts (no payment header provided)
	PaymentCount int64      `json:"payment_count"` // Track successful payments (payment for x402)
	AccessCount  int64      `json:"access_count"`  // Track successful accesses (payment header provided)
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

// Purchase records information about a successful payment transaction
type Purchase struct {
	ShortCode      string    `json:"short_code"`
	TargetURL      string    `json:"target_url"`
	Method         string    `json:"method"`
	Price          int64     `json:"price"`
	IsTest         bool      `json:"is_test"`
	PaymentPayload string    `json:"payment_payload"` // X-Payment header as JSON
	SettleResponse string    `json:"settle_response"` // Settled response as base64
	PaidRouteID    uint      `json:"-"`               // Associated PaidRoute
	PaidRoute      PaidRoute `json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
