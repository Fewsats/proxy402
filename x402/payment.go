package x402

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	x402http "github.com/coinbase/x402/go/http"
	x402types "github.com/coinbase/x402/go/types"
	"github.com/gin-gonic/gin"
)

const x402Version = 1

// PaymentOptions is the options for the Payment helper.
type PaymentOptions struct {
	Description       string
	MimeType          string
	MaxTimeoutSeconds int
	OutputSchema      *json.RawMessage
	FacilitatorURL    string
	Testnet           bool
	Resource          string
	ResourceRootURL   string
}

// Options is the type for options accepted by Payment.
type Options func(*PaymentOptions)

func WithDescription(description string) Options {
	return func(options *PaymentOptions) {
		options.Description = description
	}
}

func WithMimeType(mimeType string) Options {
	return func(options *PaymentOptions) {
		options.MimeType = mimeType
	}
}

func WithMaxTimeoutSeconds(maxTimeoutSeconds int) Options {
	return func(options *PaymentOptions) {
		options.MaxTimeoutSeconds = maxTimeoutSeconds
	}
}

func WithOutputSchema(outputSchema *json.RawMessage) Options {
	return func(options *PaymentOptions) {
		options.OutputSchema = outputSchema
	}
}

func WithFacilitatorURL(url string) Options {
	return func(options *PaymentOptions) {
		options.FacilitatorURL = url
	}
}

func WithTestnet(testnet bool) Options {
	return func(options *PaymentOptions) {
		options.Testnet = testnet
	}
}

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

// Amount: the decimal denominated amount to charge (ex: 0.01 for 1 cent).
// Returns marshaled payload and settle response JSON bytes when payment succeeds.
func Payment(c *gin.Context, amount *big.Float, address string, opts ...Options) (paymentPayloadJSON []byte, settleResponseJSON []byte) {
	options := &PaymentOptions{
		FacilitatorURL:    x402http.DefaultFacilitatorURL,
		MaxTimeoutSeconds: 60,
		Testnet:           true,
	}
	for _, opt := range opts {
		opt(options)
	}

	c.Header("Payment-Protocol", "X402")

	network := "base"
	usdcAddress := "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
	if options.Testnet {
		network = "base-sepolia"
		usdcAddress = "0x036CbD53842c5426634e7929541eC2318f3dCF7e"
	}

	var resource string
	if options.Resource == "" {
		resource = options.ResourceRootURL + c.Request.URL.Path
	} else {
		resource = options.Resource
	}

	maxAmountRequired, _ := new(big.Float).Mul(amount, big.NewFloat(1e6)).Int(nil)
	paymentRequirements := x402types.PaymentRequirementsV1{
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
	}
	if err := setUSDCInfoV1(&paymentRequirements, options.Testnet); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error":       err.Error(),
			"x402Version": x402Version,
		})
		return nil, nil
	}

	paymentRequirementsJSON, _ := json.Marshal(paymentRequirements)

	userAgent := c.GetHeader("User-Agent")
	acceptHeader := c.GetHeader("Accept")
	isWebBrowser := strings.Contains(acceptHeader, "text/html") && strings.Contains(userAgent, "Mozilla")

	headerValue := c.GetHeader("X-PAYMENT")
	if headerValue == "" {
		respondPaymentRequiredV1(c, isWebBrowser, resource, amount, options, paymentRequirements, paymentRequirementsJSON, "X-PAYMENT header is required")
		return nil, nil
	}

	decodedPayload, err := base64.StdEncoding.DecodeString(headerValue)
	if err != nil {
		respondPaymentRequiredV1(c, isWebBrowser, resource, amount, options, paymentRequirements, paymentRequirementsJSON, "invalid X-PAYMENT header")
		return nil, nil
	}
	version, err := x402types.DetectVersion(decodedPayload)
	if err != nil || version != 1 {
		respondPaymentRequiredV1(c, isWebBrowser, resource, amount, options, paymentRequirements, paymentRequirementsJSON, "invalid x402 version for v1 route")
		return nil, nil
	}

	_, err = x402types.ToPaymentPayloadV1(decodedPayload)
	if err != nil {
		respondPaymentRequiredV1(c, isWebBrowser, resource, amount, options, paymentRequirements, paymentRequirementsJSON, "failed to decode payment payload")
		return nil, nil
	}

	facilitator := x402http.NewFacilitatorClient(&x402http.FacilitatorConfig{
		URL: options.FacilitatorURL,
	})
	requirementsJSON, err := json.Marshal(paymentRequirements)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error":       err.Error(),
			"x402Version": x402Version,
		})
		return nil, nil
	}

	verifyResponse, err := facilitator.Verify(c.Request.Context(), decodedPayload, requirementsJSON)
	if err != nil || verifyResponse == nil || !verifyResponse.IsValid {
		errMsg := "invalid payment"
		if err != nil {
			errMsg = err.Error()
		} else if verifyResponse != nil && verifyResponse.InvalidReason != "" {
			errMsg = verifyResponse.InvalidReason
		}
		respondPaymentRequiredV1(c, isWebBrowser, resource, amount, options, paymentRequirements, paymentRequirementsJSON, errMsg)
		return nil, nil
	}

	settleResponse, err := facilitator.Settle(c.Request.Context(), decodedPayload, requirementsJSON)
	if err != nil || settleResponse == nil || !settleResponse.Success {
		errMsg := "payment settlement failed"
		if err != nil {
			errMsg = err.Error()
		} else if settleResponse != nil && settleResponse.ErrorReason != "" {
			errMsg = settleResponse.ErrorReason
		}
		respondPaymentRequiredV1(c, isWebBrowser, resource, amount, options, paymentRequirements, paymentRequirementsJSON, errMsg)
		return nil, nil
	}

	paymentPayloadJSON = decodedPayload
	settleResponseJSON, err = json.Marshal(settleResponse)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error":       err.Error(),
			"x402Version": x402Version,
		})
		return nil, nil
	}
	c.Header("X-PAYMENT-RESPONSE", base64.StdEncoding.EncodeToString(settleResponseJSON))

	return paymentPayloadJSON, settleResponseJSON
}

func setUSDCInfoV1(requirements *x402types.PaymentRequirementsV1, testnet bool) error {
	usdcInfo := map[string]any{
		"name":    "USDC",
		"version": "2",
	}
	if !testnet {
		usdcInfo["name"] = "USD Coin"
	}

	extraJSON, err := json.Marshal(usdcInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal USDC info: %w", err)
	}
	raw := json.RawMessage(extraJSON)
	requirements.Extra = &raw
	return nil
}

func respondPaymentRequiredV1(c *gin.Context, isWebBrowser bool, resource string, amount *big.Float, options *PaymentOptions, requirements x402types.PaymentRequirementsV1, requirementsJSON []byte, errMsg string) {
	if isWebBrowser {
		amountString := amount.Text('f', 6)
		description := options.Description
		if contextDescription := c.GetString("Description"); contextDescription != "" {
			description = contextDescription
		}

		c.HTML(http.StatusPaymentRequired, "payment_required.html", gin.H{
			"Resource":                resource,
			"Description":             description,
			"AmountFormatted":         amountString,
			"ErrorMessage":            errMsg,
			"ResourceType":            c.GetString("ResourceType"),
			"OriginalFilename":        c.GetString("OriginalFilename"),
			"Title":                   c.GetString("Title"),
			"CoverURL":                c.GetString("CoverURL"),
			"IsTestnet":               options.Testnet,
			"PaymentRequirements":     requirements,
			"PaymentRequirementsJSON": string(requirementsJSON),
		})
		c.Abort()
		return
	}

	if errMsg == "" {
		errMsg = "X-PAYMENT header is required"
	}
	c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
		"error":       errMsg,
		"accepts":     []x402types.PaymentRequirementsV1{requirements},
		"x402Version": x402Version,
	})
}
