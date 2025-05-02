package routes

import (
	"linkshrink/users"
	"time"
)

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
	User         users.User `json:"-"`
	IsEnabled    bool       `json:"is_enabled"`
	AttemptCount int64      `json:"attempt_count"` // Track payment attempts (no payment header provided)
	PaymentCount int64      `json:"payment_count"` // Track successful payments (payment for x402)
	AccessCount  int64      `json:"access_count"`  // Track successful accesses (payment header provided)
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}
