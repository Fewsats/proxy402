package ui

import (
	"embed"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"linkshrink/auth"
	"linkshrink/routes"
	"linkshrink/users"
)

// UIHandler handles UI-related routes and rendering
type UIHandler struct {
	paidRouteService *routes.PaidRouteService
	authService      *auth.Service
	userService      *users.UserService

	templatesFS embed.FS

	config *Config
	logger *slog.Logger
}

// NewUIHandler creates a new UIHandler instance
func NewUIHandler(paidRouteService *routes.PaidRouteService,
	authService *auth.Service, userService *users.UserService,
	cfg *Config, templatesFS embed.FS, logger *slog.Logger) *UIHandler {

	return &UIHandler{
		paidRouteService: paidRouteService,
		authService:      authService,
		userService:      userService,

		templatesFS: templatesFS,

		config: cfg,
		logger: logger,
	}
}

// UIPaidRoute is a UI model for displaying PaidRoute with formatted price
type UIPaidRoute struct {
	ID        uint64
	UserID    uint64
	ShortCode string
	TargetURL string
	Method    string
	Price     string

	IsTest    bool
	IsEnabled bool

	AttemptCount uint64
	PaymentCount uint64
	AccessCount  uint64
	CreatedAt    string
}

// getBaseURL returns the base URL (scheme + host) for the current request
func (h *UIHandler) getBaseURL(gCtx *gin.Context) string {
	scheme := "http"
	// Check X-Forwarded-Proto first, as we might be behind a proxy
	if proto := gCtx.GetHeader("X-Forwarded-Proto"); proto == "https" {
		scheme = "https"
	} else if gCtx.Request.TLS != nil { // Fallback to checking direct TLS connection
		scheme = "https"
	}
	return scheme + "://" + gCtx.Request.Host
}

// SetupRoutes registers UI routes to the provided router
func (h *UIHandler) SetupRoutes(router *gin.Engine) {
	// Public landing page for non-authenticated users
	router.GET("/", h.handleLandingPage)

	// Dashboard for authenticated users
	router.GET("/dashboard", auth.AuthMiddleware(h.authService), h.handleDashboard)

	// Settings page
	router.GET("/settings", auth.AuthMiddleware(h.authService), h.handleSettings)

	// Regenerate secret
	router.POST("/settings/regenerate-secret", auth.AuthMiddleware(h.authService), h.handleRegenerateSecret)

	// Update payment address
	router.POST("/settings/update-payment-address", auth.AuthMiddleware(h.authService), h.handleUpdatePaymentAddress)
}

// handleLandingPage renders the landing page for non-authenticated users
func (h *UIHandler) handleLandingPage(gCtx *gin.Context) {
	// Check if user is already authenticated via cookie
	cookie, err := gCtx.Cookie("jwt")
	if err == nil && cookie != "" {
		// User has JWT cookie, redirect to dashboard
		gCtx.Redirect(http.StatusFound, "/dashboard")
		return
	}

	// Render landing page for non-authenticated users
	gCtx.HTML(http.StatusOK, "landing.html", gin.H{
		"baseURL":           h.getBaseURL(gCtx),
		"GoogleAnalyticsID": h.config.GoogleAnalyticsID,
	})
}

// handleDashboard handles the main dashboard page for authenticated users
func (h *UIHandler) handleDashboard(gCtx *gin.Context) {
	// User is guaranteed to exist due to middleware
	userID, err := auth.GetUserID(gCtx)
	if err != nil {
		gCtx.Redirect(http.StatusFound, "/")
		return
	}

	user, err := h.userService.GetUserByID(gCtx.Request.Context(), userID)
	if err != nil {
		gCtx.HTML(http.StatusInternalServerError, "dashboard.html", gin.H{
			"error":             "Unable to fetch user details",
			"user":              user,
			"GoogleAnalyticsID": h.config.GoogleAnalyticsID,
		})
		return
	}

	// Get user's links
	dbLinks, err := h.paidRouteService.ListUserRoutes(gCtx.Request.Context(), userID)
	if err != nil {
		gCtx.HTML(http.StatusInternalServerError, "dashboard.html", gin.H{
			"error":             "Unable to fetch links",
			"user":              user,
			"GoogleAnalyticsID": h.config.GoogleAnalyticsID,
		})
		return
	}

	// Convert DB models to UI models
	var uiLinks []UIPaidRoute
	for _, link := range dbLinks {
		uiLinks = append(uiLinks, UIPaidRoute{
			ID:           link.ID,
			UserID:       link.UserID,
			ShortCode:    link.ShortCode,
			TargetURL:    link.TargetURL,
			Method:       link.Method,
			Price:        strconv.FormatFloat(float64(link.Price)/1000000, 'f', -1, 64),
			IsEnabled:    link.IsEnabled,
			AttemptCount: link.AttemptCount,
			PaymentCount: link.PaymentCount,
			AccessCount:  link.AccessCount,
			IsTest:       link.IsTest,
			CreatedAt:    link.CreatedAt.Format("2006-01-02"),
		})
	}

	baseURL := h.getBaseURL(gCtx)
	host := gCtx.Request.Host

	gCtx.HTML(http.StatusOK, "dashboard.html", gin.H{
		"user":              user,
		"links":             uiLinks,
		"host":              host,
		"baseURL":           baseURL,
		"GoogleAnalyticsID": h.config.GoogleAnalyticsID,
	})
}

// handleSettings handles the settings page
func (h *UIHandler) handleSettings(gCtx *gin.Context) {
	// User is guaranteed to exist due to middleware
	userID, err := auth.GetUserID(gCtx)
	if err != nil {
		h.logger.Error("auth user not found in context", "error", err)
		gCtx.Redirect(http.StatusFound, "/")
		return
	}

	user, err := h.userService.GetUserByID(gCtx.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Settings page error: failed to get user details",
			"userID", userID,
			"error", err)
		gCtx.HTML(http.StatusInternalServerError, "settings.html", gin.H{
			"error":             "Unable to fetch user details",
			"user":              user,
			"GoogleAnalyticsID": h.config.GoogleAnalyticsID,
		})
		return
	}

	// Pass data to the template
	gCtx.HTML(http.StatusOK, "settings.html", gin.H{
		"user":              user,
		"proxy_secret":      user.Proxy402Secret,
		"payment_address":   user.PaymentAddress,
		"GoogleAnalyticsID": h.config.GoogleAnalyticsID,
	})
}

// handleRegenerateSecret regenerates the user's proxy secret
func (h *UIHandler) handleRegenerateSecret(gCtx *gin.Context) {
	// User is guaranteed to exist due to middleware
	userID, err := auth.GetUserID(gCtx)
	if err != nil {
		gCtx.Redirect(http.StatusFound, "/")
		return
	}

	// Generate and update the secret
	user, err := h.userService.UpdateProxySecret(gCtx.Request.Context(), userID)
	if err != nil {
		gCtx.HTML(http.StatusInternalServerError, "settings.html", gin.H{
			"error":             "Failed to regenerate secret",
			"userID":            userID,
			"GoogleAnalyticsID": h.config.GoogleAnalyticsID,
		})
		return
	}

	// Return form with success message
	gCtx.HTML(http.StatusOK, "settings.html", gin.H{
		"user":              user,
		"proxy_secret":      user.Proxy402Secret,
		"payment_address":   user.PaymentAddress,
		"message":           "Secret regenerated successfully",
		"GoogleAnalyticsID": h.config.GoogleAnalyticsID,
	})
}

// handleUpdatePaymentAddress handles the update payment address form submission
func (h *UIHandler) handleUpdatePaymentAddress(gCtx *gin.Context) {
	// User is guaranteed to exist due to middleware
	userID, err := auth.GetUserID(gCtx)
	if err != nil {
		gCtx.Redirect(http.StatusFound, "/")
		return
	}

	user, err := h.userService.GetUserByID(gCtx.Request.Context(), userID)
	if err != nil {
		gCtx.HTML(http.StatusInternalServerError, "settings.html", gin.H{
			"error":             "Unable to fetch user details",
			"user":              users.User{},
			"GoogleAnalyticsID": h.config.GoogleAnalyticsID,
		})
		return
	}

	paymentAddress := gCtx.PostForm("payment_address")

	// Update payment address
	user, err = h.userService.UpdatePaymentAddress(gCtx.Request.Context(),
		userID, paymentAddress)
	if err != nil {
		gCtx.HTML(http.StatusBadRequest, "settings.html", gin.H{
			"error":             "Failed to update payment address: " + err.Error(),
			"user":              user,
			"GoogleAnalyticsID": h.config.GoogleAnalyticsID,
		})
		return
	}

	// Render updated form
	gCtx.HTML(http.StatusOK, "settings.html", gin.H{
		"user":              user,
		"message":           "Payment address updated successfully",
		"GoogleAnalyticsID": h.config.GoogleAnalyticsID,
	})
}
