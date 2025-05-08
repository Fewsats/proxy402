package ui

// Config holds the configuration for the UI package.
type Config struct {
	GoogleAnalyticsID string `long:"google_analytics_id" description:"Google Analytics Tracking ID"`
}

// DefaultConfig returns default values for the UI Config struct.
func DefaultConfig() Config {
	return Config{
		GoogleAnalyticsID: "", // Default to empty, indicating no tracking
	}
}
