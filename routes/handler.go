package routes

import (
	// Needed for potential body buffering if required later
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"

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

		config: config,
		logger: logger,
	}
}

// CreatePaidRouteRequest defines the JSON body for creating a paid route.
type CreatePaidRouteRequest struct {
	TargetURL string `json:"target_url" binding:"required,url"`
	Method    string `json:"method" binding:"required"`
	Price     string `json:"price" binding:"required,numeric"` // Validate as numeric string
	IsTest    bool   `json:"is_test" binding:"omitempty"`      // Optional, defaults to true if omitted
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

// formatPrice converts price from integer (USDC * 10^6) to a decimal string
func (h *PaidRouteHandler) formatPrice(priceInt uint64) string {
	return fmt.Sprintf("%.6f", float64(priceInt)/1000000)
}

// CreatePaidRouteHandler handles POST requests to create new paid routes.
// NOTE: Currently doesn't enforce specific auth/admin checks, assumes authenticated user.
func (h *PaidRouteHandler) CreatePaidRouteHandler(gCtx *gin.Context) {
	var req CreatePaidRouteRequest
	if err := gCtx.ShouldBindJSON(&req); err != nil {
		gCtx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Get user ID from the context (set by AuthMiddleware)
	authPayload, exists := gCtx.Get(auth.AuthorizationPayloadKey)
	if !exists {
		gCtx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	payload := authPayload.(*auth.Claims)

	// Call the service to create the route, passing isTestValue
	route, err := h.paidRouteService.CreatePaidRoute(gCtx.Request.Context(), req.TargetURL, req.Method, req.Price, req.IsTest, payload.UserID)
	if err != nil {
		// Handle specific validation errors from the service
		gCtx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Construct the full access URL using the determined scheme
	scheme := getRequestScheme(gCtx)
	accessURL := fmt.Sprintf("%s://%s/%s", scheme, gCtx.Request.Host, route.ShortCode)

	gCtx.JSON(http.StatusCreated, gin.H{
		"id":            route.ID,
		"short_code":    route.ShortCode,
		"access_url":    accessURL,
		"target_url":    route.TargetURL,
		"method":        route.Method,
		"price":         h.formatPrice(route.Price),
		"is_test":       route.IsTest,
		"is_enabled":    route.IsEnabled,
		"attempt_count": route.AttemptCount,
		"payment_count": route.PaymentCount,
		"access_count":  route.AccessCount,
		"created_at":    route.CreatedAt,
	})
}

// HandlePaidRoute handles requests to the dynamic /:shortCode endpoints.
// This performs DB lookup, method check, payment verification, and then proxies the request.
func (h *PaidRouteHandler) HandlePaidRoute(gCtx *gin.Context) {
	shortCode := gCtx.Param("shortCode")

	// Find the enabled route configuration by its short code.
	route, err := h.paidRouteService.FindEnabledRouteByShortCode(gCtx.Request.Context(), shortCode)
	if err != nil {
		if errors.Is(err, ErrRouteNotFound) {
			h.logger.Error("Route not found", "shortCode", shortCode)
			gCtx.JSON(http.StatusNotFound, gin.H{"error": "Route not found or is disabled."})
		} else {
			h.logger.Error("Error retrieving route config", "shortCode", shortCode, "error", err)
			gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving route configuration."})
		}
		return
	}

	// Check if the request method matches the configured method for the route.
	if gCtx.Request.Method != route.Method {
		gCtx.JSON(http.StatusMethodNotAllowed, gin.H{"error": fmt.Sprintf("Method %s not allowed for this route. Allowed: %s", gCtx.Request.Method, route.Method)})
		return
	}

	// --- Payment Verification Step ---
	// 3. Parse route.Price string to *big.Float
	// Convert int64 price to string and then to *big.Float
	priceFloat, ok := new(big.Float).SetString(h.formatPrice(route.Price))
	if !ok {
		h.logger.Error("Invalid price format", "shortCode", shortCode, "price", route.Price)
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal configuration error for route price."})
		return
	}

	scheme := getRequestScheme(gCtx)
	accessURL := fmt.Sprintf("%s://%s/%s", scheme, gCtx.Request.Host, route.ShortCode)
	h.logger.Info("Access URL created", "shortCode", shortCode, "accessURL", accessURL)

	// Try to get the user's payment address if they have one configured
	user, err := h.userService.GetUserByID(gCtx.Request.Context(), route.UserID)
	if err != nil {
		h.logger.Error("Error fetching user", "shortCode", shortCode, "userID", route.UserID, "error", err)
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error processing request after payment."})
	}

	paymentAddress := h.config.X402PaymentAddress // Default to configured address
	if user.PaymentAddress != "" {
		paymentAddress = user.PaymentAddress
	}

	paymentPayload, settleResponse := x402.Payment(gCtx, priceFloat, paymentAddress,
		x402.OptionWithFacilitatorURL(h.config.X402FacilitatorURL),
		x402.OptionWithTestnet(route.IsTest), // Use the value from the route
		x402.OptionWithDescription(fmt.Sprintf("Payment for %s %s", route.Method, accessURL)),
		x402.OptionWithResource(accessURL),
		// Add other options like OptionWithMaxTimeoutSeconds if needed
	)

	// 6. Check if the payment function aborted the request
	if gCtx.IsAborted() {
		// If aborted with 402, increment attempt count
		if gCtx.Writer.Status() == http.StatusPaymentRequired {
			err := h.paidRouteService.IncrementAttemptCount(gCtx.Request.Context(), shortCode)
			if err != nil {
				// Log error, but don't overwrite the original 402 response
				h.logger.Error("Error incrementing attempt count after 402", "shortCode", shortCode, "error", err)
			}
		}
		return // Stop processing, response already sent by x402.Payment
	}
	// If we get here, payment verification within x402.Payment succeeded.
	// --- END Payment Verification ---

	// Save purchase record
	h.logger.Info("Saving purchase record", "shortCode", shortCode, "routeID", route.ID)
	err = h.savePurchaseRecord(gCtx.Request.Context(), route, paymentAddress, paymentPayload, settleResponse)
	if err != nil {
		h.logger.Error("Failed to save purchase record", "shortCode", shortCode, "routeID", route.ID, "error", err)
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Increment access count *after* successful verification check
	if err := h.paidRouteService.IncrementAccessCount(gCtx.Request.Context(), shortCode); err != nil {
		// Log the error, but proceed with proxying? Or return 500?
		// Let's log and return 500 for now, as failing to count access is an internal issue.
		h.logger.Error("Failed to increment access count", "shortCode", shortCode, "error", err)
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error processing request after payment."})
		return
	}

	err = h.paidRouteService.IncrementPaymentCount(gCtx.Request.Context(), shortCode)
	if err != nil {
		h.logger.Error("Failed to increment payment count", "shortCode", shortCode, "error", err)
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error processing request after payment."})
		return
	}

	// --- Perform Reverse Proxy ---

	// 7. Parse the target URL
	h.logger.Info("Parsing target URL", "shortCode", shortCode, "targetURL", route.TargetURL)
	targetURL, err := url.Parse(route.TargetURL)
	if err != nil {
		h.logger.Error("Failed to parse target URL", "shortCode", shortCode, "targetURL", route.TargetURL, "error", err)
		gCtx.Error(fmt.Errorf("failed to parse target URL for route %s: %w", shortCode, err))
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal configuration error for route target."})
		return
	}

	// 8. Create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// 9. Define the Director to modify the request before forwarding
	originalDirector := proxy.Director // Keep original director for basic setup
	proxy.Director = func(req *http.Request) {
		originalDirector(req) // Apply default modifications (Scheme, Host, Path)
		// Ensure the Host header is set correctly for the target server
		req.Host = targetURL.Host
		// Explicitly set the path to the target path, overwriting the incoming one
		req.URL.Path = targetURL.Path

		// --- Add Proxy402-Secret Header ---
		user, err := h.userService.GetUserByID(gCtx.Request.Context(), route.UserID)
		if err != nil {
			// Log the error, but don't expose details to the client via headers.
			// The proxy error handler below will catch this if the request fails later.
			h.logger.Error("Error fetching user for Proxy402-Secret", "shortCode", shortCode, "error", err)
		} else if user != nil && user.Proxy402Secret != "" {
			req.Header.Set("Proxy402-Secret", user.Proxy402Secret)
		}
		// --- End Add Proxy402-Secret Header ---

		// Optional: Clean up headers specific to the incoming request if needed
		// req.Header.Del("X-Forwarded-For")

		// Note: The default reverse proxy handles X-Forwarded-For etc. automatically.
		// We mostly just need to ensure req.Host is correct.
		// The original gCtx.Request.URL path should be preserved by default director.
	}

	// Optional: Custom error handling
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		h.logger.Error("Reverse proxy error", "shortCode", shortCode,
			"targetURL", route.TargetURL, "error", err)

		rw.WriteHeader(http.StatusBadGateway)
		// Avoid writing detailed errors to the client unless necessary
		// rw.Write([]byte("Proxy error"))
	}

	// 10. Serve the request using the proxy
	// This forwards the request (method, headers, body) to the targetURL
	// and streams the response back to the original client (gCtx.Writer).
	proxy.ServeHTTP(gCtx.Writer, gCtx.Request)

	// --- END Reverse Proxy ---
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

	// Format the response (similar to Create response, maybe factor out a helper)
	responseRoutes := make([]gin.H, len(routes))
	for i, route := range routes {
		scheme := getRequestScheme(gCtx)
		accessURL := fmt.Sprintf("%s://%s/%s", scheme, gCtx.Request.Host, route.ShortCode)
		responseRoutes[i] = gin.H{
			"id":            route.ID,
			"short_code":    route.ShortCode,
			"access_url":    accessURL,
			"target_url":    route.TargetURL,
			"method":        route.Method,
			"price":         h.formatPrice(route.Price),
			"is_test":       route.IsTest,
			"is_enabled":    route.IsEnabled,
			"attempt_count": route.AttemptCount,
			"payment_count": route.PaymentCount,
			"access_count":  route.AccessCount,
			"created_at":    route.CreatedAt,
			"updated_at":    route.UpdatedAt,
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

// savePurchaseRecord is a helper method to save the purchase record in the database.
func (h *PaidRouteHandler) savePurchaseRecord(gCtx context.Context,
	route *PaidRoute, paymentAddress string,
	paymentPayload *x402.PaymentPayload,
	settleResponse *x402.SettleResponse) error {

	// Convert paymentPayload to JSON
	paymentPayloadJson, err := json.Marshal(paymentPayload)
	if err != nil {
		h.logger.Error("Failed to encode payment payload", "error", err)
		return fmt.Errorf("failed to encode payment payload: %w", err)
	}

	// Convert settleResponse to JSON
	settleResponseJson, err := json.Marshal(settleResponse)
	if err != nil {
		h.logger.Error("Failed to encode settle response", "error", err)
		return fmt.Errorf("failed to encode settle response: %w", err)
	}

	// Create purchase record
	purchase := &purchases.Purchase{
		ShortCode:      route.ShortCode,
		TargetURL:      route.TargetURL,
		Method:         route.Method,
		Price:          route.Price,
		IsTest:         route.IsTest,
		PaidRouteID:    route.ID,
		PaymentPayload: paymentPayloadJson,
		SettleResponse: settleResponseJson,
		PaidToAddress:  paymentAddress,
	}
	h.logger.Info("Purchase record created", "purchase", fmt.Sprintf("%+v", purchase))

	_, err = h.purchaseService.CreatePurchase(gCtx, purchase)
	if err != nil {
		h.logger.Error("Failed to save purchase record", "routeID", route.ID, "shortCode", route.ShortCode,
			"targetURL", route.TargetURL, "method", route.Method, "price", route.Price, "isTest", route.IsTest,
			"createdAt", route.CreatedAt, "updatedAt", route.UpdatedAt, "error", err)
		return fmt.Errorf("failed to save purchase record: %w", err)
	}

	return nil
}
