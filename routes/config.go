package routes

// DefaultConfig returns all default values for the Config struct.
func DefaultConfig() *Config {
	return &Config{
		X402TestnetPaymentAddress: "",
		X402MainnetPaymentAddress: "",
		X402FacilitatorURL:        "https://x402.org/facilitator",
	}
}

// Config holds the configuration for the routes package.
type Config struct {
	X402TestnetPaymentAddress string `long:"x402_testnet_payment_address" description:"Testnet payment address for X402"`
	X402MainnetPaymentAddress string `long:"x402_mainnet_payment_address" description:"Mainnet payment address for X402"`
	X402FacilitatorURL        string `long:"x402_facilitator_url" description:"URL for X402 facilitator"`
}
