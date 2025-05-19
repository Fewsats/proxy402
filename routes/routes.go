package routes

import (
	"fmt"
	"strconv"
	"time"
)

// PaidRoute represents a configurable, paid API route proxied by the service.
type PaidRoute struct {
	ID        uint64 `json:"-"`
	ShortCode string `json:"short_code"`
	Method    string `json:"method"` // GET, POST, PUT, DELETE, PATCH

	// TargetURL is the URL that the route will proxy requests to.
	// If ResourceType is "file", this will be the R2 object key.
	TargetURL string `json:"-"`
	// Resource type is "url" or "file"
	ResourceType string `json:"resource_type"`
	// Original filename is the filename of the file uploaded by the user when resource_type is "file"
	OriginalFilename *string `json:"original_filename,omitempty"`
	// CoverImageURL is the URL of the cover image uploaded by the user
	CoverImageURL *string `json:"cover_image_url,omitempty"`
	// Title of the route
	Title *string `json:"title,omitempty"`
	// Description of the route
	Description *string `json:"description,omitempty"`

	// Store price as int64 representing base units (USDC * 10^6)
	Price     uint64 `json:"price"`
	Type      string `json:"type"` // e.g., "credit", "subscription"
	Credits   uint64 `json:"credits"`
	IsTest    bool   `json:"is_test"`
	UserID    uint64 `json:"-"`
	IsEnabled bool   `json:"is_enabled"`

	AttemptCount uint64 `json:"attempt_count"` // Track payment attempts (no payment header provided)
	PaymentCount uint64 `json:"payment_count"` // Track successful payments (payment for x402)
	AccessCount  uint64 `json:"access_count"`  // Track successful accesses (payment header provided)

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// PriceUtils contains utility functions for working with prices
type PriceUtils struct{}

// FormatPrice converts price from integer (USDC * 10^6) to a decimal string
func (PriceUtils) FormatPrice(priceInt uint64) string {
	return fmt.Sprintf("%.6f", float64(priceInt)/1000000)
}

// ParsePrice converts a price string to integer units (USDC * 10^6)
// Returns the price as uint64 and any error encountered
func (PriceUtils) ParsePrice(priceStr string) (uint64, error) {
	// Parse the price string to float
	priceFloat, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid price format: must be a decimal number")
	}

	// Validate price is not negative
	if priceFloat < 0 {
		return 0, fmt.Errorf("price must be greater or equal to 0")
	}

	// Convert to integer (USDC * 10^6)
	return uint64(priceFloat * 1000000), nil
}

// NewPriceUtils creates a new PriceUtils instance
func NewPriceUtils() PriceUtils {
	return PriceUtils{}
}
