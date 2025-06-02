package ui

// Config holds the configuration for the UI package.
type Config struct {
	GoogleAnalyticsID   string `long:"google_analytics_id" description:"Google Analytics Tracking ID"`
	BetterStackToken    string `long:"better_stack_token" description:"Better Stack Token"`
	BetterStackEndpoint string `long:"better_stack_endpoint" description:"Better Stack Endpoint"`
}

// DefaultConfig returns default values for the UI Config struct.
func DefaultConfig() Config {
	return Config{
		GoogleAnalyticsID:   "", // Default to empty, indicating no tracking
		BetterStackToken:    "", // Default to empty, indicating no logging
		BetterStackEndpoint: "", // Default to empty, indicating no logging
	}
}
