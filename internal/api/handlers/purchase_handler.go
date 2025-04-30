package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"linkshrink/internal/api/middleware"
	"linkshrink/internal/auth"
	"linkshrink/internal/core/services"
	"linkshrink/internal/store"
)

// PurchaseHandler handles HTTP requests related to purchases
type PurchaseHandler struct {
	purchaseService *services.PurchaseService
}

// NewPurchaseHandler creates a new PurchaseHandler
func NewPurchaseHandler(purchaseService *services.PurchaseService) *PurchaseHandler {
	return &PurchaseHandler{
		purchaseService: purchaseService,
	}
}

// DashboardStats contains aggregated purchase data for the dashboard
type DashboardStats struct {
	TotalEarnings  int64              `json:"total_earnings"`
	TotalPurchases int                `json:"total_purchases"`
	DailyPurchases []store.DailyStats `json:"daily_purchases"`
}

// GetDashboardStats returns purchase statistics for the dashboard
func (h *PurchaseHandler) GetDashboardStats(ctx *gin.Context) {
	// Get user ID from the context (set by AuthMiddleware)
	authPayload, exists := ctx.Get(middleware.AuthorizationPayloadKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	payload := authPayload.(*auth.Claims)

	// Get dashboard stats for the last 7 days
	dailyStats, totalEarnings, totalPurchases, err := h.purchaseService.GetDashboardStats(payload.UserID, 7)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve purchase data"})
		return
	}

	stats := DashboardStats{
		TotalEarnings:  totalEarnings,
		TotalPurchases: totalPurchases,
		DailyPurchases: dailyStats,
	}

	ctx.JSON(http.StatusOK, stats)
}
