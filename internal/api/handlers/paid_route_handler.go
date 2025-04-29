package handlers

import (
	"errors" // Import errors package
	"fmt"

	// "math/big" // Not used yet
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm" // Import gorm for ErrRecordNotFound

	"linkshrink/internal/api/middleware" // To get user ID
	"linkshrink/internal/auth"           // To get Claims type

	// "linkshrink/internal/config" // Not used yet
	"linkshrink/internal/core/services"
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
// This performs DB lookup, method check, (TODO: payment verification), and redirects.
func (h *PaidRouteHandler) HandlePaidRoute(ctx *gin.Context) {
	shortCode := ctx.Param("shortCode")
	// fmt.Printf("[DEBUG] HandlePaidRoute received shortCode: '%s'\n", shortCode)

	// 1. Find route in DB
	route, err := h.paidRouteService.FindEnabledRouteByShortCode(shortCode)
	if err != nil {
		// Use errors.Is for robust check against gorm.ErrRecordNotFound
		if errors.Is(err, gorm.ErrRecordNotFound) || err.Error() == "route not found or not enabled" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Route not found or is disabled."})
		} else {
			ctx.Error(err) // Log unexpected errors
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving route configuration."})
		}
		return
	}

	// 2. Check Method (Optional for Redirects, but keep for consistency for now)
	// Note: Browsers might change method to GET on 301/302 redirects.
	// Use 307/308 for strict method preservation if needed, but GET is typical.
	if ctx.Request.Method != route.Method {
		ctx.JSON(http.StatusMethodNotAllowed, gin.H{"error": fmt.Sprintf("Method %s not allowed for this route. Allowed: %s", ctx.Request.Method, route.Method)})
		return
	}

	// --- TODO: Payment Verification Step ---
	// ... (Parsing price, calling VerifyX402Payment, etc.) ...
	// If payment fails, the verification function should handle the 402 response.
	// If payment succeeds, we continue to the redirect.
	// --- END TODO: Payment Verification ---

	// Increment payment count (Best effort after successful access/verification)
	h.paidRouteService.IncrementPaymentCount(shortCode)

	// --- Perform the Redirect ---
	// Use StatusFound (302) for temporary redirect. Use StatusMovedPermanently (301) if appropriate.
	ctx.Redirect(http.StatusFound, route.TargetURL)

	// --- Proxy code removed ---
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
