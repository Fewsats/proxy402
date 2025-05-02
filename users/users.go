package users

import (
	"time"
)

// User represents a registered user in the system.
type User struct {
	ID             uint               `json:"-"`
	Email          string             `json:"email"`                                                 // Email is the primary identifier
	Name           string             `json:"name,omitempty"`                                        // User's name from Google
	GoogleID       string             `json:"-"`                                                     // Google user ID
	PaidRoutes     []routes.PaidRoute `json:"-"`                                                     // User has many PaidRoutes
	Proxy402Secret string             `gorm:"column:proxy_402_secret;uniqueIndex;not null" json:"-"` // Secret for forwarded request verification

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
