package auth

import (
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// DefaultConfig returns all default values for the Config struct.
func DefaultConfig() *Config {
	return &Config{
		JWTSecret:          "insecure-jwt-secret",
		JWTExpirationHours: 72 * time.Hour,
		GoogleClientID:     "",
		GoogleClientSecret: "",
		GoogleRedirectURL:  "",
	}
}

// Config holds the configuration for the auth package.
type Config struct {
	JWTSecret          string        `long:"jwt_secret" description:"Secret used to sign JWT tokens"`
	JWTExpirationHours time.Duration `long:"jwt_expiration_hours" description:"JWT token expiration time in hours"`
	GoogleClientID     string        `long:"google_client_id" description:"Google OAuth client ID"`
	GoogleClientSecret string        `long:"google_client_secret" description:"Google OAuth client secret"`
	GoogleRedirectURL  string        `long:"google_redirect_url" description:"Google OAuth redirect URL"`
}

// GetGoogleOAuthConfig returns the OAuth2 config for Google
func (c *Config) GetGoogleOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     c.GoogleClientID,
		ClientSecret: c.GoogleClientSecret,
		RedirectURL:  c.GoogleRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}
