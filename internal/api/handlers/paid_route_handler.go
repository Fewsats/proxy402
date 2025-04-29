package handlers

import (
	"errors" // Import errors package
	"fmt"
	"math/big" // Need big.Float for price parsing
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm" // Import gorm for ErrRecordNotFound

	"linkshrink/internal/api/middleware" // To get user ID
	"linkshrink/internal/auth"           // To get Claims type
	"linkshrink/internal/config"         // Need for X402 config
	"linkshrink/internal/core/services"
	"linkshrink/internal/x402" // Import local x402 package
	// "linkshrink/internal/x402" // Import local x402 when verification is added
)

// PaidRouteHandler handles HTTP requests related to paid routes.
type PaidRouteHandler struct {
	paidRouteService *services.PaidRouteService
	// We might need linkService later if we want to avoid shortCode collisions
}

// NewPaidRouteHandler creates a new PaidRouteHandler.
func NewPaidRouteHandler(service *services.PaidRouteService) *PaidRouteHandler {
	return &PaidRouteHandler{paidRouteService: service}
}

// CreatePaidRouteRequest defines the JSON body for creating a paid route.
type CreatePaidRouteRequest struct {
	TargetURL string `json:"target_url" binding:"required,url"`
	Method    string `json:"method" binding:"required"`
	Price     string `json:"price" binding:"required,numeric"` // Validate as numeric string
	// Asset field removed as per user request
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

	// Call the service to create the route
	route, err := h.paidRouteService.CreatePaidRoute(req.TargetURL, req.Method, req.Price, payload.UserID)
	if err != nil {
		// Handle specific validation errors from the service
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Construct the full access URL
	accessURL := fmt.Sprintf("http://%s/%s", ctx.Request.Host, route.ShortCode)

	ctx.JSON(http.StatusCreated, gin.H{
		"id":            route.ID,
		"short_code":    route.ShortCode,
		"access_url":    accessURL,
		"target_url":    route.TargetURL,
		"method":        route.Method,
		"price":         h.formatPrice(route.Price),
		"is_enabled":    route.IsEnabled,
		"attempt_count": route.AttemptCount,
		"payment_count": route.PaymentCount,
		"access_count":  route.AccessCount,
		"created_at":    route.CreatedAt,
	})
}

// HandlePaidRoute handles requests to the dynamic /:shortCode endpoints.
// This performs DB lookup, method check, payment verification, and redirects.
func (h *PaidRouteHandler) HandlePaidRoute(ctx *gin.Context) {
	shortCode := ctx.Param("shortCode")
	// fmt.Printf("[DEBUG] HandlePaidRoute received shortCode: '%s'\n", shortCode)

	// 1. Find route in DB
	route, err := h.paidRouteService.FindEnabledRouteByShortCode(shortCode)
	if err != nil {
		// Use errors.Is for robust check against gorm.ErrRecordNotFound
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Route not found or is disabled."})
		} else {
			ctx.Error(err) // Log unexpected errors
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving route configuration."})
		}
		return
	}

	// 2. Check Method
	if ctx.Request.Method != route.Method {
		ctx.JSON(http.StatusMethodNotAllowed, gin.H{"error": fmt.Sprintf("Method %s not allowed for this route. Allowed: %s", ctx.Request.Method, route.Method)})
		return
	}

	// --- Payment Verification Step ---

	// 3. Parse route.Price string to *big.Float
	priceFloat, ok := new(big.Float).SetString(route.Price)
	if !ok {
		ctx.Error(fmt.Errorf("invalid price format stored for route %s: %s", shortCode, route.Price))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal configuration error for route price."})
		return
	}

	// 4. Prepare options for the x402.Payment function
	// Construct the resource URL the user is actually accessing
	accessURL := fmt.Sprintf("http://%s/%s", ctx.Request.Host, route.ShortCode)

	// Note: The x402 package option functions are now exported (start with uppercase).
	// We can call them directly.

	// 5. Call x402.Payment function (treating it as a pre-handler check)
	x402.Payment(ctx, priceFloat, config.AppConfig.X402PaymentAddress,
		// Use the exported option functions (uppercase)
		x402.OptionWithFacilitatorURL(config.AppConfig.X402FacilitatorURL),
		x402.OptionWithTestnet(true), // TODO: Make configurable
		x402.OptionWithDescription(fmt.Sprintf("Payment for %s %s", route.Method, accessURL)),
		x402.OptionWithResource(accessURL),
		// Add other options like OptionWithMaxTimeoutSeconds if needed
	)

	// 6. Check if the payment function aborted the request
	if ctx.IsAborted() {
		fmt.Printf("Payment check failed or required for %s, request aborted by x402.Payment\n", shortCode)
		return // Stop processing, response already sent by x402.Payment
	}
	// If we get here, payment verification within x402.Payment presumably succeeded and called ctx.Next()
	// --- END Payment Verification ---

	// Increment payment count only after successful verification check
	h.paidRouteService.IncrementPaymentCount(shortCode)

	// --- Perform the Redirect ---
	ctx.Redirect(http.StatusFound, route.TargetURL)
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
		accessURL := fmt.Sprintf("http://%s/%s", ctx.Request.Host, route.ShortCode)
		responseRoutes[i] = gin.H{
			"id":            route.ID,
			"short_code":    route.ShortCode,
			"access_url":    accessURL,
			"target_url":    route.TargetURL,
			"method":        route.Method,
			"price":         h.formatPrice(route.Price),
			"is_enabled":    route.IsEnabled,
			"attempt_count": route.AttemptCount,
			"payment_count": route.PaymentCount,
			"access_count":  route.AccessCount,
			"created_at":    route.CreatedAt,
			"updated_at":    route.UpdatedAt, // Include updated_at for listing
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
