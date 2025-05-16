package x402

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/coinbase/x402/go/pkg/facilitatorclient"
	"github.com/coinbase/x402/go/pkg/types"
	"github.com/gin-gonic/gin"
)

const x402Version = 1

// PaymentOptions is the options for the PaymentMiddleware.
type PaymentOptions struct {
	Description       string
	MimeType          string
	MaxTimeoutSeconds int
	OutputSchema      *json.RawMessage
	FacilitatorConfig *types.FacilitatorConfig
	Testnet           bool
	Resource          string
	ResourceRootURL   string
}

// Options is the type for the options for the PaymentMiddleware.
type Options func(*PaymentOptions)

// WithDescription is an option for the PaymentMiddleware to set the description.
func WithDescription(description string) Options {
	return func(options *PaymentOptions) {
		options.Description = description
	}
}

// WithMimeType is an option for the PaymentMiddleware to set the mime type.
func WithMimeType(mimeType string) Options {
	return func(options *PaymentOptions) {
		options.MimeType = mimeType
	}
}

// WithMaxDeadlineSeconds is an option for the PaymentMiddleware to set the max timeout seconds.
func WithMaxTimeoutSeconds(maxTimeoutSeconds int) Options {
	return func(options *PaymentOptions) {
		options.MaxTimeoutSeconds = maxTimeoutSeconds
	}
}

// WithOutputSchema is an option for the PaymentMiddleware to set the output schema.
func WithOutputSchema(outputSchema *json.RawMessage) Options {
	return func(options *PaymentOptions) {
		options.OutputSchema = outputSchema
	}
}

// WithFacilitatorConfig is an option for the PaymentMiddleware to set the facilitator config.
func WithFacilitatorConfig(config *types.FacilitatorConfig) Options {
	return func(options *PaymentOptions) {
		options.FacilitatorConfig = config
	}
}

// WithTestnet is an option for the PaymentMiddleware to set the testnet flag.
func WithTestnet(testnet bool) Options {
	return func(options *PaymentOptions) {
		options.Testnet = testnet
	}
}

// WithResource is an option for the PaymentMiddleware to set the resource.
func WithResource(resource string) Options {
	return func(options *PaymentOptions) {
		options.Resource = resource
	}
}

func WithResourceRootURL(resourceRootURL string) Options {
	return func(options *PaymentOptions) {
		options.ResourceRootURL = resourceRootURL
	}
}

// Amount: the decimal denominated amount to charge (ex: 0.01 for 1 cent)
func Payment(c *gin.Context, amount *big.Float, address string, opts ...Options) (paymentPayload *types.PaymentPayload, settleResponse *types.SettleResponse) {
	options := &PaymentOptions{
		FacilitatorConfig: &types.FacilitatorConfig{
			URL: facilitatorclient.DefaultFacilitatorURL,
		},
		MaxTimeoutSeconds: 60,
		Testnet:           true,
	}

	for _, opt := range opts {
		opt(options)
	}

	// Set the Payment-Protocol header to support other payment protocols
	c.Header("Payment-Protocol", "X402")

	var (
		network              = "base"
		usdcAddress          = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
		facilitatorClient    = facilitatorclient.NewFacilitatorClient(options.FacilitatorConfig)
		maxAmountRequired, _ = new(big.Float).Mul(amount, big.NewFloat(1e6)).Int(nil)
	)

	if options.Testnet {
		network = "base-sepolia"
		usdcAddress = "0x036CbD53842c5426634e7929541eC2318f3dCF7e"
	}

	fmt.Println("Payment middleware checking request:", c.Request.URL)

	userAgent := c.GetHeader("User-Agent")
	acceptHeader := c.GetHeader("Accept")
	isWebBrowser := strings.Contains(acceptHeader, "text/html") && strings.Contains(userAgent, "Mozilla")

	var resource string
	if options.Resource == "" {
		resource = options.ResourceRootURL + c.Request.URL.Path
	} else {
		resource = options.Resource
	}

	paymentRequirements := &types.PaymentRequirements{
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
			"error":       err.Error(),
			"x402Version": x402Version,
		})
		return
	}

	payment := c.GetHeader("X-PAYMENT")
	paymentPayload, err := types.DecodePaymentPayloadFromBase64(payment)
	if err != nil {
		// For browser requests, always serve HTML template
		if isWebBrowser {
			// Format the amount for display (convert from big.Float to string)
			amountString := amount.Text('f', 6)

			c.HTML(http.StatusPaymentRequired, "payment_required.html", gin.H{
				"Resource":         resource,
				"Description":      options.Description,
				"AmountFormatted":  amountString,
				"ResourceType":     c.GetString("ResourceType"),
				"OriginalFilename": c.GetString("OriginalFilename"),
			})
			c.Abort()
			return
		}

		// For API clients, return JSON
		c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
			"error":       "X-PAYMENT header is required",
			"accepts":     []*types.PaymentRequirements{paymentRequirements},
			"x402Version": x402Version,
		})
		return
	}
	paymentPayload.X402Version = x402Version

	// Verify payment
	response, err := facilitatorClient.Verify(paymentPayload, paymentRequirements)
	if err != nil {
		fmt.Println("failed to verify", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error":       err.Error(),
			"x402Version": x402Version,
		})
		return
	}

	if !response.IsValid {
		fmt.Println("Invalid payment: ", response.InvalidReason)

		// For invalid payments from browsers, always serve HTML template
		if isWebBrowser {
			// Format the amount for display with 6 decimal places
			amountString := amount.Text('f', 6)

			c.HTML(http.StatusPaymentRequired, "payment_required.html", gin.H{
				"Resource":         resource,
				"Description":      options.Description,
				"AmountFormatted":  amountString,
				"ErrorMessage":     response.InvalidReason,
				"ResourceType":     c.GetString("ResourceType"),
				"OriginalFilename": c.GetString("OriginalFilename"),
			})
			c.Abort()
			return
		}

		c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
			"error":       response.InvalidReason,
			"accepts":     []*types.PaymentRequirements{paymentRequirements},
			"x402Version": x402Version,
		})
		return
	}

	fmt.Println("Payment verified, proceeding")

	// Settle payment
	settleResponse, err = facilitatorClient.Settle(paymentPayload, paymentRequirements)
	if err != nil {
		fmt.Println("x402 Abort: Settlement failed:", err)
		fmt.Println("Settlement failed:", err)

		// For settlement errors in browsers, always serve HTML template
		if isWebBrowser {
			// Format the amount for display with 6 decimal places
			amountString := amount.Text('f', 6)

			c.HTML(http.StatusPaymentRequired, "payment_required.html", gin.H{
				"Resource":         resource,
				"Description":      options.Description,
				"AmountFormatted":  amountString,
				"ErrorMessage":     err.Error(),
				"ResourceType":     c.GetString("ResourceType"),
				"OriginalFilename": c.GetString("OriginalFilename"),
			})
			c.Abort()
			return
		}

		c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
			"error":       err.Error(),
			"accepts":     []*types.PaymentRequirements{paymentRequirements},
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
		return
	}

	c.Header("X-PAYMENT-RESPONSE", settleResponseHeader)
	return
}
