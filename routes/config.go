package routes

// DefaultConfig returns all default values for the Config struct.
func DefaultConfig() *Config {
	return &Config{
		X402PaymentAddress:    "",
		X402FacilitatorURL:    "https://x402.org/facilitator",
		X402MaxTimeoutSeconds: 300,
	}
}

// Config holds the configuration for the routes package.
type Config struct {
	X402PaymentAddress    string `long:"x402_payment_address" description:"Payment address for X402"`
	X402FacilitatorURL    string `long:"x402_facilitator_url" description:"URL for X402 facilitator"`
	X402MaxTimeoutSeconds int    `long:"x402_max_timeout_seconds" description:"Max timeout seconds for X402"`
	CDPAPIKeyID           string `long:"cdp_api_key_id" description:"API key ID for CDP"`
	CDPAPIKeySecret       string `long:"cdp_api_key_secret" description:"API key secret for CDP"`
}
