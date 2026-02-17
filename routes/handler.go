package routes

import (
	// Needed for potential body buffering if required later
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	x402core "github.com/coinbase/x402/go"
	x402http "github.com/coinbase/x402/go/http"
	x402gin "github.com/coinbase/x402/go/http/gin"
	x402evmserver "github.com/coinbase/x402/go/mechanisms/evm/exact/server"
	"github.com/gin-gonic/gin"

	"linkshrink/auth"
	"linkshrink/purchases"
	"linkshrink/users"
	"linkshrink/x402"
)

// PaidRouteHandler handles HTTP requests related to paid routes.
type PaidRouteHandler struct {
	paidRouteService *PaidRouteService
	purchaseService  *purchases.PurchaseService
	userService      *users.UserService
	priceUtils       PriceUtils

	config *Config
	logger *slog.Logger
}

// NewPaidRouteHandler creates a new PaidRouteHandler.
func NewPaidRouteHandler(routeService *PaidRouteService,
	purchaseService *purchases.PurchaseService, userService *users.UserService,
	config *Config, logger *slog.Logger) *PaidRouteHandler {

	return &PaidRouteHandler{
		paidRouteService: routeService,
		purchaseService:  purchaseService,
		userService:      userService,
		priceUtils:       NewPriceUtils(),

		config: config,
		logger: logger,
	}
}

// getRequestScheme determines the scheme (http/https) based on the request.
func getRequestScheme(gCtx *gin.Context) string {
	scheme := "http"
	if proto := gCtx.GetHeader("X-Forwarded-Proto"); proto == "https" {
		scheme = "https"
	} else if gCtx.Request.TLS != nil {
		scheme = "https"
	}
	return scheme
}

func getRoutePaymentVersion(route *PaidRoute) uint16 {
	version := route.PaymentProtocolVersion
	if version == 0 {
		return PaymentProtocolVersionV1
	}
	return version
}

func getPaymentHeaderNameForVersion(version uint16) string {
	if version == PaymentProtocolVersionV2 {
		return "PAYMENT-SIGNATURE"
	}
	return "X-Payment"
}

func getPaymentHeaderNameForRoute(route *PaidRoute) string {
	return getPaymentHeaderNameForVersion(getRoutePaymentVersion(route))
}

// setDefaultTypeAndCredits sets default values for Type and Credits if not provided.
func setDefaultTypeAndCredits(routeType *string, credits *uint64) {
	if *routeType == "" {
		*routeType = "credit"
	}
	if *credits == 0 {
		*credits = 1
	}
}

// CreatePaidRouteRequest defines the JSON body for creating a paid route.
type CreatePaidRouteRequest struct {
	TargetURL   string                `form:"target_url" binding:"required,url"`
	Method      string                `form:"method" binding:"required"`
	Price       string                `form:"price" binding:"required,numeric"` // Validate as numeric string
	IsTest      bool                  `form:"is_test" binding:"omitempty"`      // Optional, defaults to true if omitted
	Type        string                `form:"type" binding:"omitempty"`         // Optional, defaults to "credit"
	Credits     uint64                `form:"credits" binding:"omitempty"`      // Optional, defaults to 1
	CoverImage  *multipart.FileHeader `form:"cover_image" binding:"omitempty"`
	Title       string                `form:"title" binding:"omitempty"`       // Optional title
	Description string                `form:"description" binding:"omitempty"` // Optional description
}

// Validate performs business rule validation for the route creation request.
func (r *CreatePaidRouteRequest) Validate() error {
	// Validate Method
	upperMethod := strings.ToUpper(r.Method)
	validMethods := map[string]bool{
		"GET":    true,
		"POST":   true,
		"PUT":    true,
		"DELETE": true,
		"PATCH":  true,
	}
	if !validMethods[upperMethod] {
		return errors.New("invalid HTTP method provided")
	}

	// Validate Price - we just need validation, not the conversion
	priceUtils := NewPriceUtils()
	_, err := priceUtils.ParsePrice(r.Price)
	return err
}

// CreateURLRouteHandler handles POST requests to create new URL paid routes.
// NOTE: Currently doesn't enforce specific auth/admin checks, assumes authenticated user.
func (h *PaidRouteHandler) CreateURLRouteHandler(gCtx *gin.Context) {
	var req CreatePaidRouteRequest
	err := gCtx.ShouldBind(&req)
	if err != nil {
		gCtx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	setDefaultTypeAndCredits(&req.Type, &req.Credits)

	err = req.Validate()
	if err != nil {
		gCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authPayload, exists := gCtx.Get(auth.AuthorizationPayloadKey)
	if !exists {
		gCtx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	payload := authPayload.(*auth.Claims)

	route, err := h.paidRouteService.CreateURLRoute(gCtx.Request.Context(), &req, payload.UserID)
	if err != nil {
		// Handle specific validation errors from the service
		gCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Construct the full access URL using the determined scheme
	scheme := getRequestScheme(gCtx)
	accessURL := fmt.Sprintf("%s://%s/%s", scheme, gCtx.Request.Host, route.ShortCode)

	response := RouteResponse{
		ID:           route.ID,
		ShortCode:    route.ShortCode,
		Target:       route.TargetURL,
		Method:       route.Method,
		ResourceType: "url", // This is always "url" for this handler

		AccessURL: accessURL,

		Price:                  h.priceUtils.FormatPrice(route.Price),
		Type:                   route.Type,
		Credits:                route.Credits,
		PaymentProtocolVersion: route.PaymentProtocolVersion,
		IsTest:                 route.IsTest,
		IsEnabled:              route.IsEnabled,

		Title:       route.Title,
		Description: route.Description,
		CoverURL:    route.CoverImageURL,

		AttemptCount: route.AttemptCount,
		PaymentCount: route.PaymentCount,
		AccessCount:  route.AccessCount,

		CreatedAt: route.CreatedAt,
		UpdatedAt: route.UpdatedAt,
	}

	gCtx.JSON(http.StatusCreated, response)
}

// CreateFileRouteRequest defines the JSON body for creating a paid route for file uploads.
type CreateFileRouteRequest struct {
	OriginalFilename string                `form:"original_filename" binding:"required"`
	Price            string                `form:"price" binding:"required,numeric"`
	IsTest           bool                  `form:"is_test" binding:"omitempty"`
	Type             string                `form:"type" binding:"omitempty"`    // Optional, defaults to "credit"
	Credits          uint64                `form:"credits" binding:"omitempty"` // Optional, defaults to 1
	CoverImage       *multipart.FileHeader `form:"cover_image" binding:"omitempty"`
	Title            string                `form:"title" binding:"omitempty"`       // Optional title
	Description      string                `form:"description" binding:"omitempty"` // Optional description
}

// Validate performs business rule validation for the file route creation request.
func (r *CreateFileRouteRequest) Validate() error {
	// Validate Price - we just need validation, not the conversion
	priceUtils := NewPriceUtils()
	_, err := priceUtils.ParsePrice(r.Price)
	return err
}

// FileRouteResponse represents a response to creating a file route.
type FileRouteResponse struct {
	UploadURL string `json:"upload_url"`
}

// CreateFileRouteHandler handles POST requests to create a paid route
// for file uploads.
// The method does not upload the file itself, it just creates the SQL route
// and returns a presigned R2 upload URL for the client to upload the file to.
func (h *PaidRouteHandler) CreateFileRouteHandler(gCtx *gin.Context) {
	var req CreateFileRouteRequest
	err := gCtx.ShouldBind(&req)
	if err != nil {
		gCtx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	setDefaultTypeAndCredits(&req.Type, &req.Credits)

	err = req.Validate()
	if err != nil {
		gCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authPayload, exists := gCtx.Get(auth.AuthorizationPayloadKey)
	if !exists {
		gCtx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	payload := authPayload.(*auth.Claims)

	_, uploadURL, err := h.paidRouteService.CreateFileRoute(
		gCtx.Request.Context(),
		&req,
		payload.UserID,
	)
	if err != nil {
		gCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gCtx.JSON(http.StatusCreated, FileRouteResponse{
		UploadURL: uploadURL,
	})
}

// getAndValidateRoute retrieves the paid route configuration based on the shortCode
// from the request path and validates the request method.
// It sends an error response and returns shouldReturn=true if validation fails or route is not found.
func (h *PaidRouteHandler) getAndValidateRoute(gCtx *gin.Context) (*PaidRoute, bool) {
	shortCode := gCtx.Param("shortCode")

	route, err := h.paidRouteService.FindEnabledRouteByShortCode(gCtx.Request.Context(), shortCode)
	if err != nil {
		if errors.Is(err, ErrRouteNotFound) {
			h.logger.Error("Route not found", "shortCode", shortCode)
			gCtx.JSON(http.StatusNotFound, gin.H{"error": "Route not found or is disabled."})
		} else {
			h.logger.Error("Error retrieving route config", "shortCode", shortCode, "error", err)
			gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving route configuration."})
		}
		return nil, true
	}

	if gCtx.Request.Method != route.Method {
		h.logger.Warn("Method not allowed for route",
			"shortCode", shortCode,
			"requestMethod", gCtx.Request.Method,
			"allowedMethod", route.Method)
		gCtx.JSON(http.StatusMethodNotAllowed, gin.H{"error": fmt.Sprintf("Method %s not allowed for this route. Allowed: %s", gCtx.Request.Method, route.Method)})
		return nil, true // Indicate main handler should return
	}

	return route, false // Route is valid, continue
}

// tryExistingPayment checks if a client-provided X-Payment header corresponds to a valid, usable purchase.
// It attempts to use a credit from such a purchase.
// Returns:
//   - usedExistingCredit: true if an existing purchase was found and a credit was successfully used.
//   - proceedToNewPayment: true if a new payment flow should be initiated (e.g., no header, purchase not found, no credits, error using credit).
func (h *PaidRouteHandler) tryExistingPayment(gCtx *gin.Context, route *PaidRoute) (usedExistingCredit bool, proceedToNewPayment bool) {
	paymentHeaderName := getPaymentHeaderNameForRoute(route)
	clientPaymentHeader := gCtx.GetHeader(paymentHeaderName)
	if clientPaymentHeader == "" {
		// No header, so must proceed to new payment flow.
		return false, true
	}

	h.logger.Debug("Client provided payment header, checking for existing purchase",
		"shortCode", route.ShortCode, "routeID", route.ID,
		"paymentHeaderName", paymentHeaderName, "clientPaymentHeader", clientPaymentHeader)

	existingPurchase, err := h.purchaseService.GetPurchaseByRouteIDAndPaymentHeader(gCtx.Request.Context(), route.ID, clientPaymentHeader)

	if err != nil {
		if !errors.Is(err, purchases.ErrPurchaseNotFound) {
			// Log actual errors, but still proceed to new payment as a fallback.
			h.logger.Debug("Error checking for existing purchase with payment header",
				"shortCode", route.ShortCode, "paymentHeader", clientPaymentHeader, "error", err)
		}
		// If purchase not found or other error, proceed to new payment.
		return false, true
	}

	if existingPurchase == nil { // Should be covered by ErrPurchaseNotFound, but as a safeguard.
		return false, true
	}

	h.logger.Debug("Existing purchase record found for payment header",
		"shortCode", route.ShortCode, "purchaseID", existingPurchase.ID,
		"creditsUsed", existingPurchase.CreditsUsed, "creditsAvailable", existingPurchase.CreditsAvailable)

	if existingPurchase.CreditsUsed >= existingPurchase.CreditsAvailable {
		h.logger.Info("Existing purchase (via header) has no credits left. Proceeding to new payment.",
			"shortCode", route.ShortCode, "purchaseID", existingPurchase.ID)
		return false, true // No credits left, proceed to new payment.
	}

	// Credits are available, attempt to use one.
	h.logger.Debug("Existing purchase has available credits. Attempting to use one credit.",
		"shortCode", route.ShortCode, "purchaseID", existingPurchase.ID)

	errIncrement := h.purchaseService.IncrementCreditsUsed(gCtx.Request.Context(), existingPurchase.ID)
	if errIncrement != nil {
		h.logger.Error("Failed to increment credits_used for existing purchase. Proceeding to new payment.",
			"shortCode", route.ShortCode, "purchaseID", existingPurchase.ID, "error", errIncrement)
		return false, true // Failed to use credit, proceed to new payment.
	}

	// Successfully used a credit from an existing purchase.
	h.logger.Info("Successfully used a credit from existing purchase via payment header.",
		"shortCode", route.ShortCode, "purchaseID", existingPurchase.ID)

	// Increment overall access count for the route.
	if err := h.paidRouteService.IncrementAccessCount(gCtx.Request.Context(), route.ShortCode); err != nil {
		h.logger.Error("Failed to increment access count (existing payment with credit used)",
			"shortCode", route.ShortCode, "purchaseID", existingPurchase.ID, "error", err)
		// Log error but still consider existing credit used successfully for proxying.
	}

	return true, false // Existing credit used, DO NOT proceed to new payment.
}

// executeNewPaymentFlow handles the entire process of a new payment if an existing one isn't used.
// Returns:
//   - paymentProcessedSuccessfully: true if a new payment was completed and purchase recorded.
//   - requestHandled: true if a response was sent (e.g., 402, 500 error) and the main handler should stop.
func (h *PaidRouteHandler) executeNewPaymentFlow(gCtx *gin.Context, route *PaidRoute) (paymentProcessedSuccessfully bool, requestHandled bool) {
	version := getRoutePaymentVersion(route)

	switch version {
	case PaymentProtocolVersionV1:
		return h.executeNewPaymentFlowV1(gCtx, route)
	case PaymentProtocolVersionV2:
		return h.executeNewPaymentFlowV2(gCtx, route)
	default:
		h.logger.Error("Unsupported payment protocol version",
			"shortCode", route.ShortCode, "routeID", route.ID, "paymentProtocolVersion", version)
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Unsupported payment protocol version"})
		return false, true
	}
}

func (h *PaidRouteHandler) executeNewPaymentFlowV1(gCtx *gin.Context, route *PaidRoute) (paymentProcessedSuccessfully bool, requestHandled bool) {
	// Parse route.Price string to *big.Float
	priceFloat, ok := new(big.Float).SetString(h.priceUtils.FormatPrice(route.Price))
	if !ok {
		h.logger.Error("Invalid price format in executeNewPaymentFlow", "shortCode", route.ShortCode, "price", route.Price)
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal configuration error for route price."})
		return false, true // Request handled (error sent)
	}

	scheme := getRequestScheme(gCtx)
	accessURL := fmt.Sprintf("%s://%s/%s", scheme, gCtx.Request.Host, route.ShortCode)
	h.logger.Debug("Access URL for new payment flow created", "shortCode", route.ShortCode, "accessURL", accessURL)

	user, err := h.userService.GetUserByID(gCtx.Request.Context(), route.UserID)
	if err != nil {
		h.logger.Error("Error fetching user in executeNewPaymentFlow", "shortCode", route.ShortCode, "userID", route.UserID, "error", err)
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error processing request for payment."})
		return false, true // Request handled (error sent)
	}

	paymentAddress := h.config.X402PaymentAddress // Default to configured address
	if user.PaymentAddress != "" {
		paymentAddress = user.PaymentAddress
	}

	// Set resource type and original filename in the context for use in payment templates
	gCtx.Set("ResourceType", route.ResourceType)
	if route.ResourceType == "file" && route.OriginalFilename != nil {
		gCtx.Set("OriginalFilename", *route.OriginalFilename)
	}

	// Set metadata fields for social sharing in payment template
	if route.Title != nil {
		gCtx.Set("Title", *route.Title)
	}
	if route.Description != nil {
		gCtx.Set("Description", *route.Description)
	}
	if route.CoverImageURL != nil {
		gCtx.Set("CoverURL", *route.CoverImageURL)
	}

	paymentPayloadJSON, settleResponseJSON := x402.Payment(gCtx, priceFloat, paymentAddress,
		x402.WithFacilitatorURL(h.config.X402FacilitatorURL),
		x402.WithDescription(fmt.Sprintf("Payment for %s %s", route.Method, accessURL)),
		x402.WithResource(accessURL),
		x402.WithTestnet(route.IsTest),
		x402.WithMaxTimeoutSeconds(h.config.X402MaxTimeoutSeconds),
	)

	if gCtx.IsAborted() {
		if gCtx.Writer.Status() == http.StatusPaymentRequired {
			h.logger.Info("Payment required (402) by x402.Payment", "shortCode", route.ShortCode)
			if err := h.paidRouteService.IncrementAttemptCount(gCtx.Request.Context(), route.ShortCode); err != nil {
				h.logger.Error("Error incrementing attempt count after 402", "shortCode", route.ShortCode, "error", err)
			}
		}
		// If x402.Payment aborted (e.g. sent 402), the request is handled.
		return false, true
	}

	// If we get here, payment verification within x402.Payment succeeded.
	paymentHeaderForNewPurchase := gCtx.GetHeader(getPaymentHeaderNameForRoute(route))
	err = h.savePurchaseRecordJSON(gCtx.Request.Context(), route, paymentAddress, paymentPayloadJSON, settleResponseJSON, paymentHeaderForNewPurchase)
	if err != nil {
		h.logger.Error("Failed to save purchase record after new payment", "shortCode", route.ShortCode, "routeID", route.ID, "error", err)
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error after payment."})
		return false, true // Request handled (error sent)
	}

	// Increment payment count as a new payment was successfully processed and saved.
	if err := h.paidRouteService.IncrementPaymentCount(gCtx.Request.Context(), route.ShortCode); err != nil {
		h.logger.Error("Failed to increment payment count after new payment", "shortCode", route.ShortCode, "error", err)
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error after processing payment counts."})
		return true, true // Payment was processed, but this is a critical error for accounting, request is handled.
	}

	// Increment access count as a new payment was made and purchase recorded.
	if err := h.paidRouteService.IncrementAccessCount(gCtx.Request.Context(), route.ShortCode); err != nil {
		h.logger.Error("Failed to increment access count after new payment", "shortCode", route.ShortCode, "error", err)
		// Log error, but consider payment successful for proxying. This count is secondary for the current access.
		// gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error after processing access counts."})
		// return true, true // If we want to stop here
	}

	return true, false // New payment processed successfully, request not yet fully handled (proxying is next).
}

func (h *PaidRouteHandler) executeNewPaymentFlowV2(gCtx *gin.Context, route *PaidRoute) (paymentProcessedSuccessfully bool, requestHandled bool) {
	scheme := getRequestScheme(gCtx)
	accessURL := fmt.Sprintf("%s://%s/%s", scheme, gCtx.Request.Host, route.ShortCode)

	user, err := h.userService.GetUserByID(gCtx.Request.Context(), route.UserID)
	if err != nil {
		h.logger.Error("Error fetching user in executeNewPaymentFlowV2", "shortCode", route.ShortCode, "userID", route.UserID, "error", err)
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error processing request for payment."})
		return false, true
	}

	paymentAddress := h.config.X402PaymentAddress
	if user.PaymentAddress != "" {
		paymentAddress = user.PaymentAddress
	}

	network := x402core.Network("eip155:8453")
	if route.IsTest {
		network = x402core.Network("eip155:84532")
	}

	routesConfig := x402http.RoutesConfig{
		"*": {
			Accepts: []x402http.PaymentOption{
				{
					Scheme:            "exact",
					PayTo:             paymentAddress,
					Price:             h.priceUtils.FormatPrice(route.Price),
					Network:           network,
					MaxTimeoutSeconds: h.config.X402MaxTimeoutSeconds,
				},
			},
			Resource:    accessURL,
			Description: fmt.Sprintf("Payment for %s %s", route.Method, accessURL),
		},
	}

	facilitatorClient := x402http.NewFacilitatorClient(&x402http.FacilitatorConfig{
		URL: h.config.X402FacilitatorURL,
	})
	server := x402http.NewServer(routesConfig,
		x402core.WithFacilitatorClient(facilitatorClient),
		x402core.WithSchemeServer(network, x402evmserver.NewExactEvmScheme()),
	)

	err = server.Initialize(gCtx.Request.Context())
	if err != nil {
		h.logger.Error("Failed to initialize x402 v2 server", "shortCode", route.ShortCode, "error", err)
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to initialize payment server"})
		return false, true
	}

	processResult := server.ProcessHTTPRequest(gCtx.Request.Context(), x402http.HTTPRequestContext{
		Adapter: x402gin.NewGinAdapter(gCtx),
		Path:    gCtx.Request.URL.Path,
		Method:  gCtx.Request.Method,
	}, nil)

	switch processResult.Type {
	case x402http.ResultPaymentError:
		if processResult.Response == nil {
			gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Unexpected payment response"})
			return false, true
		}

		for key, value := range processResult.Response.Headers {
			gCtx.Header(key, value)
		}

		if processResult.Response.IsHTML {
			body, ok := processResult.Response.Body.(string)
			if !ok {
				gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid paywall response"})
				return false, true
			}
			gCtx.Data(processResult.Response.Status, "text/html; charset=utf-8", []byte(body))
		} else if processResult.Response.Body == nil {
			gCtx.Status(processResult.Response.Status)
		} else {
			gCtx.JSON(processResult.Response.Status, processResult.Response.Body)
		}

		if processResult.Response.Status == http.StatusPaymentRequired {
			errIncrement := h.paidRouteService.IncrementAttemptCount(gCtx.Request.Context(), route.ShortCode)
			if errIncrement != nil {
				h.logger.Error("Error incrementing attempt count after v2 402", "shortCode", route.ShortCode, "error", errIncrement)
			}
		}

		return false, true

	case x402http.ResultPaymentVerified:
		// Continue below.
	case x402http.ResultNoPaymentRequired:
		h.logger.Error("v2 route did not require payment as expected", "shortCode", route.ShortCode, "routeID", route.ID)
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal payment configuration error"})
		return false, true
	default:
		h.logger.Error("Unknown x402 v2 payment result", "shortCode", route.ShortCode, "resultType", processResult.Type)
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal payment processing error"})
		return false, true
	}

	if processResult.PaymentPayload == nil || processResult.PaymentRequirements == nil {
		h.logger.Error("v2 payment verification succeeded without payload/requirements", "shortCode", route.ShortCode)
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal payment verification error"})
		return false, true
	}

	settleResponse, err := server.SettlePayment(gCtx.Request.Context(), *processResult.PaymentPayload, *processResult.PaymentRequirements)
	if err != nil {
		h.logger.Error("v2 settlement failed", "shortCode", route.ShortCode, "error", err)
		gCtx.JSON(http.StatusPaymentRequired, gin.H{"error": err.Error(), "x402Version": 2})
		return false, true
	}

	settleResponseJSON, err := json.Marshal(settleResponse)
	if err != nil {
		h.logger.Error("Failed to encode v2 settle response", "shortCode", route.ShortCode, "error", err)
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error after payment settlement."})
		return false, true
	}
	gCtx.Header("PAYMENT-RESPONSE", base64.StdEncoding.EncodeToString(settleResponseJSON))

	paymentPayloadJSON, err := json.Marshal(processResult.PaymentPayload)
	if err != nil {
		h.logger.Error("Failed to encode v2 payment payload", "shortCode", route.ShortCode, "error", err)
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error after payment verification."})
		return false, true
	}

	paymentHeaderForNewPurchase := gCtx.GetHeader(getPaymentHeaderNameForRoute(route))
	err = h.savePurchaseRecordJSON(gCtx.Request.Context(), route, paymentAddress, paymentPayloadJSON, settleResponseJSON, paymentHeaderForNewPurchase)
	if err != nil {
		h.logger.Error("Failed to save purchase record after v2 payment", "shortCode", route.ShortCode, "routeID", route.ID, "error", err)
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error after payment."})
		return false, true
	}

	err = h.paidRouteService.IncrementPaymentCount(gCtx.Request.Context(), route.ShortCode)
	if err != nil {
		h.logger.Error("Failed to increment payment count after v2 payment", "shortCode", route.ShortCode, "error", err)
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error after processing payment counts."})
		return true, true
	}

	err = h.paidRouteService.IncrementAccessCount(gCtx.Request.Context(), route.ShortCode)
	if err != nil {
		h.logger.Error("Failed to increment access count after v2 payment", "shortCode", route.ShortCode, "error", err)
	}

	return true, false
}

// proxyRequest sets up and executes the reverse proxy to the target URL.
func (h *PaidRouteHandler) proxyRequest(gCtx *gin.Context, route *PaidRoute) {
	h.logger.Debug("Attempting to proxy request",
		"shortCode", route.ShortCode,
		"targetURL", route.TargetURL,
		"resourceType", route.ResourceType)

	// Handle file resources differently - detect by ResourceType or URL format
	if route.ResourceType == "file" || !strings.Contains(route.TargetURL, "://") {
		h.logger.Debug("Detected file resource, generating presigned URL",
			"shortCode", route.ShortCode,
			"r2Key", route.TargetURL)

		var originalFilenameToShow string
		if route.OriginalFilename != nil {
			originalFilenameToShow = *route.OriginalFilename
		} else {
			// This case should not happen for ResourceType == "file"
			h.logger.Warn("File route is missing OriginalFilename", "shortCode", route.ShortCode, "r2Key", route.TargetURL)
			originalFilenameToShow = route.TargetURL // use r2 key as fallback
		}

		// Generate a presigned download URL
		downloadURL, err := h.paidRouteService.GetFileDownloadURL(gCtx.Request.Context(), route.TargetURL, originalFilenameToShow)
		if err != nil {
			h.logger.Error("Failed to generate presigned download URL",
				"shortCode", route.ShortCode, "r2Key", route.TargetURL, "error", err)
			gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate download link"})
			return
		}

		// Return the URL as JSON for programmatic clients
		gCtx.JSON(http.StatusOK, gin.H{
			"download_url": downloadURL,
			"filename":     originalFilenameToShow,
		})
		return
	}

	// Normal URL proxy for resource_type = "url"
	targetURL, err := url.Parse(route.TargetURL)
	if err != nil {
		h.logger.Error("Failed to parse target URL for proxy",
			"shortCode", route.ShortCode, "targetURL", route.TargetURL, "error", err)
		// gCtx.Error() might have already been called by a previous step if this was part of it
		// However, if this is the first time parsing it in a refactored flow, an error response is needed.
		if !gCtx.IsAborted() { // Avoid double writing headers
			gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal configuration error for route target."})
		}
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = targetURL.Host
		req.URL.Path = targetURL.Path // Explicitly set the path to the target path

		// Add Proxy402-Secret Header if configured for the user
		user, err := h.userService.GetUserByID(gCtx.Request.Context(), route.UserID)
		if err != nil {
			h.logger.Error("Error fetching user for Proxy402-Secret in proxy director",
				"shortCode", route.ShortCode, "userID", route.UserID, "error", err)
		} else if user != nil && user.Proxy402Secret != "" {
			req.Header.Set("Proxy402-Secret", user.Proxy402Secret)
		}
	}

	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		h.logger.Error("Reverse proxy error occurred",
			"shortCode", route.ShortCode, "targetURL", route.TargetURL, "error", err)
		// Avoid writing detailed errors to the client unless necessary
		// Check if headers have already been written to avoid http.ErrHeaderSent panic
		if !gCtx.Writer.Written() {
			rw.WriteHeader(http.StatusBadGateway)
		}
	}

	proxy.ServeHTTP(gCtx.Writer, gCtx.Request)
}

// HandlePaidRoute handles requests to the dynamic /:shortCode endpoints.
func (h *PaidRouteHandler) HandlePaidRoute(gCtx *gin.Context) {
	route, shouldReturn := h.getAndValidateRoute(gCtx)
	if shouldReturn {
		return
	}

	usedExistingCredit, proceedToNewPayment := h.tryExistingPayment(gCtx, route)

	var newPaymentProcessedSuccessfully bool = false

	if proceedToNewPayment {
		var requestAlreadyHandled bool
		newPaymentProcessedSuccessfully, requestAlreadyHandled = h.executeNewPaymentFlow(gCtx, route)
		if requestAlreadyHandled {
			return
		}
	}

	if !usedExistingCredit && !newPaymentProcessedSuccessfully {
		if !gCtx.IsAborted() {
			h.logger.Error("Logical error: Reached proxy stage without a clear payment path and context not aborted.", "shortCode", route.ShortCode)
			gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server processing error."})
		}
		return
	}

	h.proxyRequest(gCtx, route)
}

// RouteResponse represents a standardized response for route data
type RouteResponse struct {
	ID           uint64 `json:"id"`
	ShortCode    string `json:"short_code"`
	Target       string `json:"target"`
	Method       string `json:"method"`
	ResourceType string `json:"resource_type"`

	AccessURL string `json:"access_url"`

	Title                  *string `json:"title,omitempty"`
	Description            *string `json:"description,omitempty"`
	CoverURL               *string `json:"cover_url,omitempty"`
	Price                  string  `json:"price"`
	Type                   string  `json:"type"`
	Credits                uint64  `json:"credits"`
	PaymentProtocolVersion uint16  `json:"payment_protocol_version"`

	IsTest    bool `json:"is_test"`
	IsEnabled bool `json:"is_enabled"`

	AttemptCount uint64 `json:"attempt_count"`
	PaymentCount uint64 `json:"payment_count"`
	AccessCount  uint64 `json:"access_count"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetUserPaidRoutes handles GET requests to retrieve all paid routes for the authenticated user.
func (h *PaidRouteHandler) GetUserPaidRoutes(gCtx *gin.Context) {
	// Get user ID from the context
	authPayload, exists := gCtx.Get(auth.AuthorizationPayloadKey)
	if !exists {
		gCtx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	payload := authPayload.(*auth.Claims)

	routes, err := h.paidRouteService.ListUserRoutes(gCtx.Request.Context(), payload.UserID)
	if err != nil {
		gCtx.Error(err)
		gCtx.JSON(http.StatusInternalServerError,
			gin.H{"error": "Failed to retrieve user routes"})

		return
	}

	// Format the response using RouteResponse struct
	responseRoutes := make([]RouteResponse, len(routes))
	for i, route := range routes {
		scheme := getRequestScheme(gCtx)
		accessURL := fmt.Sprintf("%s://%s/%s", scheme, gCtx.Request.Host, route.ShortCode)

		// Determine what to show as "target" based on resource type
		var target string
		if route.ResourceType == "file" && route.OriginalFilename != nil {
			target = *route.OriginalFilename
		} else {
			target = route.TargetURL
		}

		responseRoutes[i] = RouteResponse{
			ID:           route.ID,
			ShortCode:    route.ShortCode,
			Target:       target,
			Method:       route.Method,
			ResourceType: route.ResourceType,

			AccessURL: accessURL,

			Title:                  route.Title,
			Description:            route.Description,
			CoverURL:               route.CoverImageURL,
			Price:                  h.priceUtils.FormatPrice(route.Price),
			Type:                   route.Type,
			Credits:                route.Credits,
			PaymentProtocolVersion: route.PaymentProtocolVersion,

			IsTest:    route.IsTest,
			IsEnabled: route.IsEnabled,

			AttemptCount: route.AttemptCount,
			PaymentCount: route.PaymentCount,
			AccessCount:  route.AccessCount,

			CreatedAt: route.CreatedAt,
			UpdatedAt: route.UpdatedAt,
		}
	}

	gCtx.JSON(http.StatusOK, responseRoutes)
}

// DeleteUserPaidRoute handles DELETE requests to delete a specific paid route.
func (h *PaidRouteHandler) DeleteUserPaidRoute(gCtx *gin.Context) {
	// Get route ID from path parameter (still named linkID in the route definition)
	routeIDStr := gCtx.Param("linkID") // IMPORTANT: Route param name mismatch potential
	routeID, err := strconv.ParseUint(routeIDStr, 10, 64)
	if err != nil {
		gCtx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid route ID format"})
		return
	}

	// Get user ID from the context
	authPayload, exists := gCtx.Get(auth.AuthorizationPayloadKey)
	if !exists {
		gCtx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	payload := authPayload.(*auth.Claims)

	err = h.paidRouteService.DeleteRoute(gCtx.Request.Context(), routeID, payload.UserID)
	if err != nil {
		if errors.Is(err, ErrRouteNoPermission) {
			gCtx.JSON(http.StatusForbidden,
				gin.H{"error": "Route not found or you do not have permission to delete it"})
		} else if errors.Is(err, ErrRouteNotFound) {
			gCtx.JSON(http.StatusNotFound, gin.H{"error": "Route not found"})
		} else {
			gCtx.Error(err)
			gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete route"})
		}
		return
	}

	// Return 200 OK with empty body instead of 204 No Content
	// This allows htmx to perform the swap and remove the element
	gCtx.Status(http.StatusOK)
}

func (h *PaidRouteHandler) savePurchaseRecordJSON(gCtx context.Context,
	route *PaidRoute, paymentAddress string,
	paymentPayloadJSON []byte,
	settleResponseJSON []byte, paymentHeader string) error {

	// Create purchase record
	purchase := &purchases.Purchase{
		ShortCode:        route.ShortCode,
		TargetURL:        route.TargetURL,
		Method:           route.Method,
		Price:            route.Price,
		Type:             route.Type,
		CreditsAvailable: route.Credits,
		CreditsUsed:      1, // If we create a purchase record, we know it's a new payment. (1 use already)
		IsTest:           route.IsTest,

		PaidRouteID:   route.ID,
		PaidToAddress: paymentAddress,

		PaymentHeader:  paymentHeader,
		PaymentPayload: paymentPayloadJSON,
		SettleResponse: settleResponseJSON,
	}
	h.logger.Info("Purchase record created", "purchase", fmt.Sprintf("%+v", purchase))

	_, err := h.purchaseService.CreatePurchase(gCtx, purchase)
	if err != nil {
		h.logger.Error("Failed to save purchase record", "routeID", route.ID, "shortCode", route.ShortCode,
			"targetURL", route.TargetURL, "method", route.Method, "price", route.Price, "isTest", route.IsTest,
			"createdAt", route.CreatedAt, "updatedAt", route.UpdatedAt, "error", err)
		return fmt.Errorf("failed to save purchase record: %w", err)
	}

	return nil
}
