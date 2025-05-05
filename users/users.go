package users

import (
	"time"
)

// User represents a registered user in the system.
type User struct {
	ID             uint   `json:"-"`
	Email          string `json:"email"`                     // Email is the primary identifier
	Name           string `json:"name,omitempty"`            // User's name from Google
	GoogleID       string `json:"-"`                         // Google user ID
	Proxy402Secret string `json:"-"`                         // Secret for forwarded request verification
	PaymentAddress string `json:"payment_address,omitempty"` // User's custom payment address

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
