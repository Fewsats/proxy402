package routes

// DefaultConfig returns all default values for the Config struct.
func DefaultConfig() *Config {
	return &Config{
		X402PaymentAddress: "",
		X402FacilitatorURL: "https://x402.org/facilitator",
	}
}

// Config holds the configuration for the routes package.
type Config struct {
	X402PaymentAddress string `long:"x402_payment_address" description:"Payment address for X402"`
	X402FacilitatorURL string `long:"x402_facilitator_url" description:"URL for X402 facilitator"`
}
