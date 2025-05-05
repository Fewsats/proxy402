package routes

import (
	"time"
)

// PaidRoute represents a configurable, paid API route proxied by the service.
type PaidRoute struct {
	ID        uint64 `json:"-"`
	ShortCode string `json:"short_code"`
	TargetURL string `json:"-"`
	Method    string `json:"method"` // GET, POST, PUT, DELETE, PATCH
	// Store price as int64 representing base units (USDC * 10^6)
	Price        uint64     `json:"price"`
	IsTest       bool       `json:"is_test"`
	UserID       uint64     `json:"-"`
	IsEnabled    bool       `json:"is_enabled"`
	AttemptCount uint64     `json:"attempt_count"` // Track payment attempts (no payment header provided)
	PaymentCount uint64     `json:"payment_count"` // Track successful payments (payment for x402)
	AccessCount  uint64     `json:"access_count"`  // Track successful accesses (payment header provided)
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}
