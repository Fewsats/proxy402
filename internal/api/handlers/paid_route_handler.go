package handlers

import (
	// Needed for potential body buffering if required later
	"encoding/json"
	"errors" // Import errors package
	"fmt"
	"math/big" // Need big.Float for price parsing
	"net/http"
	"net/http/httputil" // Import for Reverse Proxy
	"net/url"           // Import for URL parsing
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm" // Import gorm for ErrRecordNotFound

	"linkshrink/internal/api/middleware" // To get user ID
	"linkshrink/internal/auth"           // To get Claims type
	"linkshrink/internal/config"         // Need for X402 config
	"linkshrink/internal/core/models"
	"linkshrink/internal/core/services"
	"linkshrink/internal/x402" // Import local x402 package
)

// PaidRouteHandler handles HTTP requests related to paid routes.
type PaidRouteHandler struct {
	paidRouteService *services.PaidRouteService
	purchaseService  *services.PurchaseService
	// We might need linkService later if we want to avoid shortCode collisions
}

// NewPaidRouteHandler creates a new PaidRouteHandler.
func NewPaidRouteHandler(routeService *services.PaidRouteService, purchaseService *services.PurchaseService) *PaidRouteHandler {
	return &PaidRouteHandler{
		paidRouteService: routeService,
		purchaseService:  purchaseService,
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
func getRequestScheme(ctx *gin.Context) string {
	scheme := "http"
	if proto := ctx.GetHeader("X-Forwarded-Proto"); proto == "https" {
		scheme = "https"
	} else if ctx.Request.TLS != nil {
		scheme = "https"
	}
	return scheme
}

// formatPrice converts price from integer (USDC * 10^6) to a decimal string
func (h *PaidRouteHandler) formatPrice(priceInt int64) string {
	return fmt.Sprintf("%.6f", float64(priceInt)/1000000)
}

// CreatePaidRouteHandler handles POST requests to create new paid routes.
// NOTE: Currently doesn't enforce specific auth/admin checks, assumes authenticated user.
func (h *PaidRouteHandler) CreatePaidRouteHandler(ctx *gin.Context) {
	var req CreatePaidRouteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Get user ID from the context (set by AuthMiddleware)
	authPayload, exists := ctx.Get(middleware.AuthorizationPayloadKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	payload := authPayload.(*auth.Claims)

	// Call the service to create the route, passing isTestValue
	route, err := h.paidRouteService.CreatePaidRoute(req.TargetURL, req.Method, req.Price, req.IsTest, payload.UserID)
	if err != nil {
		// Handle specific validation errors from the service
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Construct the full access URL using the determined scheme
	scheme := getRequestScheme(ctx)
	accessURL := fmt.Sprintf("%s://%s/%s", scheme, ctx.Request.Host, route.ShortCode)

	ctx.JSON(http.StatusCreated, gin.H{
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
func (h *PaidRouteHandler) HandlePaidRoute(ctx *gin.Context) {
	shortCode := ctx.Param("shortCode")

	// Find the enabled route configuration by its short code.
	route, err := h.paidRouteService.FindEnabledRouteByShortCode(shortCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Route not found or is disabled."})
		} else {
			fmt.Printf("Error retrieving route config for %s: %v\n", shortCode, err) // Log internal error
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving route configuration."})
		}
		return
	}

	// Check if the request method matches the configured method for the route.
	if ctx.Request.Method != route.Method {
		ctx.JSON(http.StatusMethodNotAllowed, gin.H{"error": fmt.Sprintf("Method %s not allowed for this route. Allowed: %s", ctx.Request.Method, route.Method)})
		return
	}

	// --- Payment Verification Step ---
	// 3. Parse route.Price string to *big.Float
	// Convert int64 price to string and then to *big.Float
	priceFloat, ok := new(big.Float).SetString(h.formatPrice(route.Price))
	if !ok {
		ctx.Error(fmt.Errorf("invalid price format stored for route %s: %d", shortCode, route.Price))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal configuration error for route price."})
		return
	}

	scheme := getRequestScheme(ctx)
	accessURL := fmt.Sprintf("%s://%s/%s", scheme, ctx.Request.Host, route.ShortCode)

	// Select payment address based on IsTest flag
	paymentAddress := config.AppConfig.X402TestnetPaymentAddress // Default to testnet (renamed variable)
	if !route.IsTest {
		paymentAddress = config.AppConfig.X402MainnetPaymentAddress
	}

	paymentPayload, settleResponse := x402.Payment(ctx, priceFloat, paymentAddress, // Use the selected address
		x402.OptionWithFacilitatorURL(config.AppConfig.X402FacilitatorURL),
		x402.OptionWithTestnet(route.IsTest), // Use the value from the route
		x402.OptionWithDescription(fmt.Sprintf("Payment for %s %s", route.Method, accessURL)),
		x402.OptionWithResource(accessURL),
		// Add other options like OptionWithMaxTimeoutSeconds if needed
	)

	// 6. Check if the payment function aborted the request
	if ctx.IsAborted() {
		fmt.Printf("Payment check failed or required for %s, request aborted by x402.Payment\n", shortCode)
		// If aborted with 402, increment attempt count
		if ctx.Writer.Status() == http.StatusPaymentRequired {
			err := h.paidRouteService.IncrementAttemptCount(shortCode)
			if err != nil {
				// Log error, but don't overwrite the original 402 response
				fmt.Printf("Error incrementing attempt count for %s after 402: %v\n", shortCode, err)
			}
		}
		return // Stop processing, response already sent by x402.Payment
	}
	// If we get here, payment verification within x402.Payment succeeded.
	// --- END Payment Verification ---

	// Save purchase record
	h.savePurchaseRecord(route, paymentPayload, settleResponse)

	// Increment access count *after* successful verification check
	if err := h.paidRouteService.IncrementAccessCount(shortCode); err != nil {
		// Log the error, but proceed with proxying? Or return 500?
		// Let's log and return 500 for now, as failing to count access is an internal issue.
		ctx.Error(fmt.Errorf("failed to increment access count for %s after successful payment verification: %w", shortCode, err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error processing request after payment."})
		return
	}

	h.paidRouteService.IncrementPaymentCount(shortCode)

	// --- Perform Reverse Proxy ---

	// 7. Parse the target URL
	targetURL, err := url.Parse(route.TargetURL)
	if err != nil {
		ctx.Error(fmt.Errorf("failed to parse target URL for route %s: %w", shortCode, err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal configuration error for route target."})
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

		// Optional: Clean up headers specific to the incoming request if needed
		// req.Header.Del("X-Forwarded-For")

		// Note: The default reverse proxy handles X-Forwarded-For etc. automatically.
		// We mostly just need to ensure req.Host is correct.
		// The original ctx.Request.URL path should be preserved by default director.
	}

	// Optional: Custom error handling
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		fmt.Printf("Reverse proxy error for %s to %s: %v\n", shortCode, route.TargetURL, err)
		rw.WriteHeader(http.StatusBadGateway)
		// Avoid writing detailed errors to the client unless necessary
		// rw.Write([]byte("Proxy error"))
	}

	// 10. Serve the request using the proxy
	// This forwards the request (method, headers, body) to the targetURL
	// and streams the response back to the original client (ctx.Writer).
	fmt.Printf("Proxying request for %s to %s\n", shortCode, route.TargetURL)
	proxy.ServeHTTP(ctx.Writer, ctx.Request)

	// --- END Reverse Proxy ---
}

// Helper function to copy headers, excluding hop-by-hop headers - REMOVED (No longer needed for redirect)
/*
func copyHeaders(src http.Header, dst http.Header) {
    // ... implementation ...
}
*/

// GetUserPaidRoutes handles GET requests to retrieve all paid routes for the authenticated user.
func (h *PaidRouteHandler) GetUserPaidRoutes(ctx *gin.Context) {
	// Get user ID from the context
	authPayload, exists := ctx.Get(middleware.AuthorizationPayloadKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	payload := authPayload.(*auth.Claims)

	routes, err := h.paidRouteService.ListUserRoutes(payload.UserID)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user routes"})
		return
	}

	// Format the response (similar to Create response, maybe factor out a helper)
	responseRoutes := make([]gin.H, len(routes))
	for i, route := range routes {
		scheme := getRequestScheme(ctx)
		accessURL := fmt.Sprintf("%s://%s/%s", scheme, ctx.Request.Host, route.ShortCode)
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

	ctx.JSON(http.StatusOK, responseRoutes)
}

// DeleteUserPaidRoute handles DELETE requests to delete a specific paid route.
func (h *PaidRouteHandler) DeleteUserPaidRoute(ctx *gin.Context) {
	// Get route ID from path parameter (still named linkID in the route definition)
	routeIDStr := ctx.Param("linkID") // IMPORTANT: Route param name mismatch potential
	routeID, err := strconv.ParseUint(routeIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid route ID format"})
		return
	}

	// Get user ID from the context
	authPayload, exists := ctx.Get(middleware.AuthorizationPayloadKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	payload := authPayload.(*auth.Claims)

	err = h.paidRouteService.DeleteRoute(uint(routeID), payload.UserID)
	if err != nil {
		if err.Error() == "route not found or you do not have permission to delete it" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			ctx.Error(err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete route"})
		}
		return
	}

	ctx.Status(http.StatusNoContent) // Success, no content to return
}

// savePurchaseRecord asynchronously saves a purchase record to the database
func (h *PaidRouteHandler) savePurchaseRecord(route *models.PaidRoute, paymentPayload *x402.PaymentPayload, settleResponse *x402.SettleResponse) {
	go func() {
		// Convert payment payload to JSON string
		paymentData, err := json.Marshal(paymentPayload)
		if err != nil {
			fmt.Printf("Failed to encode payment payload: %v\n", err)
			return
		}

		// Get settle response as encoded string
		settleData, err := json.Marshal(settleResponse)
		if err != nil {
			fmt.Printf("Failed to encode settle response: %v\n", err)
			return
		}

		// Save purchase info
		err = h.purchaseService.SavePurchase(
			route.ShortCode,
			route.TargetURL,
			route.Method,
			route.Price,
			route.ID,
			string(paymentData),
			string(settleData),
		)

		if err != nil {
			fmt.Printf("Failed to save purchase record: %v\n", err)
		}
	}()
}
