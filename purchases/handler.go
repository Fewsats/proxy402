package purchases

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"linkshrink/auth"
)

// PurchaseHandler handles HTTP requests related to purchases
type PurchaseHandler struct {
	purchaseService *PurchaseService
}

// NewPurchaseHandler creates a new PurchaseHandler
func NewPurchaseHandler(purchaseService *PurchaseService) *PurchaseHandler {
	return &PurchaseHandler{
		purchaseService: purchaseService,
	}
}

// DashboardStats contains aggregated purchase data for the dashboard
type DashboardStats struct {
	TotalEarnings  uint64 `json:"total_earnings"`
	TotalPurchases uint64 `json:"total_purchases"`

	TestEarnings   uint64       `json:"test_earnings"`
	TestPurchases  uint64       `json:"test_purchases"`
	RealEarnings   uint64       `json:"real_earnings"`
	RealPurchases  uint64       `json:"real_purchases"`
	DailyPurchases []DailyStats `json:"daily_purchases"`
}

// GetDashboardStats returns purchase statistics for the dashboard
func (h *PurchaseHandler) GetDashboardStats(gCtx *gin.Context) {
	// Get user ID from the context (set by AuthMiddleware)
	authPayload, exists := gCtx.Get(auth.AuthorizationPayloadKey)
	if !exists {
		gCtx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	payload := authPayload.(*auth.Claims)

	// Get dashboard stats for the last 7 days
	dailyStats, err := h.purchaseService.GetDashboardStats(gCtx.Request.Context(), payload.UserID, 7)
	if err != nil {
		gCtx.Error(err)
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve purchase data"})
		return
	}

	// Calculate test vs. real totals
	var testEarnings, realEarnings uint64
	var testPurchases, realPurchases uint64

	for _, day := range dailyStats {
		testEarnings += day.TestEarnings
		testPurchases += day.TestCount
		realEarnings += day.RealEarnings
		realPurchases += day.RealCount
	}

	stats := DashboardStats{
		TestEarnings:   testEarnings,
		TestPurchases:  testPurchases,
		RealEarnings:   realEarnings,
		RealPurchases:  realPurchases,
		DailyPurchases: dailyStats,
	}

	gCtx.JSON(http.StatusOK, stats)
}
