package x402

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	"github.com/gin-gonic/gin"
)

// PaymentMiddlewareOptions is the options for the PaymentMiddleware.
type PaymentOptions struct {
	Description       string
	MimeType          string
	MaxTimeoutSeconds int
	OutputSchema      *json.RawMessage
	FacilitatorURL    string
	Testnet           bool
	CustomPaywallHTML string
	Resource          string
	ResourceRootURL   string
}

// Options is the type for the options for the PaymentMiddleware.
type x402Options func(*PaymentOptions)

// WithDescription is an option for the PaymentMiddleware to set the description.
func OptionWithDescription(description string) x402Options {
	return func(options *PaymentOptions) {
		options.Description = description
	}
}

// WithMimeType is an option for the PaymentMiddleware to set the mime type.
func OptionWithMimeType(mimeType string) x402Options {
	return func(options *PaymentOptions) {
		options.MimeType = mimeType
	}
}

// WithMaxDeadlineSeconds is an option for the PaymentMiddleware to set the max timeout seconds.
func OptionWithMaxTimeoutSeconds(maxTimeoutSeconds int) x402Options {
	return func(options *PaymentOptions) {
		options.MaxTimeoutSeconds = maxTimeoutSeconds
	}
}

// WithOutputSchema is an option for the PaymentMiddleware to set the output schema.
func OptionWithOutputSchema(outputSchema *json.RawMessage) x402Options {
	return func(options *PaymentOptions) {
		options.OutputSchema = outputSchema
	}
}

// WithFacilitatorURL is an option for the PaymentMiddleware to set the facilitator URL.
func OptionWithFacilitatorURL(facilitatorURL string) x402Options {
	return func(options *PaymentOptions) {
		options.FacilitatorURL = facilitatorURL
	}
}

// WithTestnet is an option for the PaymentMiddleware to set the testnet flag.
func OptionWithTestnet(testnet bool) x402Options {
	return func(options *PaymentOptions) {
		options.Testnet = testnet
	}
}

// WithCustomPaywallHTML is an option for the PaymentMiddleware to set the custom paywall HTML.
func OptionWithCustomPaywallHTML(customPaywallHTML string) x402Options {
	return func(options *PaymentOptions) {
		options.CustomPaywallHTML = customPaywallHTML
	}
}

// WithResource is an option for the PaymentMiddleware to set the resource.
func OptionWithResource(resource string) x402Options {
	return func(options *PaymentOptions) {
		options.Resource = resource
	}
}

// WithResourceRootURL is an option for the PaymentMiddleware to set the resource root URL.
func OptionWithResourceRootURL(resourceRootURL string) x402Options {
	return func(options *PaymentOptions) {
		options.ResourceRootURL = resourceRootURL
	}
}

// Amount: the decimal denominated amount to charge (ex: 0.01 for 1 cent)
func Payment(c *gin.Context, amount *big.Float, address string, opts ...x402Options) {
	options := &PaymentOptions{
		FacilitatorURL:    DefaultFacilitatorURL,
		MaxTimeoutSeconds: 60,
		Testnet:           true,
	}

	for _, opt := range opts {
		opt(options)
	}

	var (
		network              = "base"
		usdcAddress          = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
		facilitatorClient    = NewFacilitatorClient(options.FacilitatorURL)
		maxAmountRequired, _ = new(big.Float).Mul(amount, big.NewFloat(1e6)).Int(nil)
	)

	if options.Testnet {
		network = "base-sepolia"
		usdcAddress = "0x036CbD53842c5426634e7929541eC2318f3dCF7e"
	}

	fmt.Println("Payment middleware checking request:", c.Request.URL)

	var resource string
	if options.Resource == "" {
		resource = options.ResourceRootURL + c.Request.URL.Path
	} else {
		resource = options.Resource
	}

	paymentRequirements := &PaymentRequirements{
		Scheme:            "exact",
		Network:           network,
		MaxAmountRequired: maxAmountRequired.String(),
		Resource:          resource,
		Description:       options.Description,
		MimeType:          options.MimeType,
		PayTo:             address,
		MaxTimeoutSeconds: options.MaxTimeoutSeconds,
		Asset:             usdcAddress,
		OutputSchema:      options.OutputSchema,
		Extra:             nil,
	}

	if err := paymentRequirements.SetUSDCInfo(options.Testnet); err != nil {
		fmt.Println("failed to set USDC info:", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}

	payment := c.GetHeader("X-PAYMENT")
	paymentPayload, err := DecodePaymentPayloadFromBase64(payment)
	if err != nil {
		fmt.Println("x402 Abort: Failed to decode X-PAYMENT header:", err)
		c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
			"error":       "X-PAYMENT header is required",
			"accepts":     []*PaymentRequirements{paymentRequirements},
			"x402Version": 1,
		})
		return
	}

	// Verify payment
	response, err := facilitatorClient.Verify(paymentPayload, paymentRequirements)
	if err != nil {
		fmt.Println("x402 Abort: Facilitator Verify call failed:", err)
		fmt.Println("failed to verify", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	if !response.IsValid {
		fmt.Println("x402 Abort: Payment verification failed. Reason:", response.InvalidReason)
		fmt.Println("Invalid payment: ", response.InvalidReason)
		c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
			"error":       response.InvalidReason,
			"accepts":     []*PaymentRequirements{paymentRequirements},
			"x402Version": 1,
		})
		return
	}

	fmt.Println("Payment verified, proceeding")
	c.Next()

	// Settle payment
	settleResponse, err := facilitatorClient.Settle(paymentPayload, paymentRequirements)
	if err != nil {
		fmt.Println("x402 Abort: Settlement failed:", err)
		fmt.Println("Settlement failed:", err)
		c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
			"error":       err.Error(),
			"accepts":     []*PaymentRequirements{paymentRequirements},
			"x402Version": 1,
		})
		return
	}

	settleResponseHeader, err := settleResponse.EncodeToBase64String()
	if err != nil {
		fmt.Println("Settle Header Encoding failed:", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}

	c.Header("X-PAYMENT-RESPONSE", settleResponseHeader)
}
