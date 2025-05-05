package ui

import (
	"embed"
	"fmt"
	"html/template"
	"log"
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

	logger *slog.Logger
}

// NewUIHandler creates a new UIHandler instance
func NewUIHandler(paidRouteService *routes.PaidRouteService,
	authService *auth.Service, userService *users.UserService, templatesFS embed.FS, logger *slog.Logger) *UIHandler {

	return &UIHandler{
		paidRouteService: paidRouteService,
		authService:      authService,
		userService:      userService,

		templatesFS: templatesFS,

		logger: logger,
	}
}

// UIPaidRoute is a UI model for displaying PaidRoute with formatted price
type UIPaidRoute struct {
	ID           uint
	UserID       uint
	ShortCode    string
	TargetURL    string
	Method       string
	Price        string
	IsEnabled    bool
	AttemptCount int64
	PaymentCount int64
	AccessCount  int64
	CreatedAt    string
	IsTest       bool
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
	// Parse HTML templates
	tmpl, err := template.ParseFS(h.templatesFS, "templates/*.html")
	if err != nil {
		log.Fatalf("failed to parse HTML templates: %v", err)
	}
	router.SetHTMLTemplate(tmpl)

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
		"baseURL": h.getBaseURL(gCtx),
	})
}

// handleDashboard handles the main dashboard page for authenticated users
func (h *UIHandler) handleDashboard(gCtx *gin.Context) {
	// User is guaranteed to exist due to middleware
	user, exists := gCtx.Get(auth.UserKey)
	if !exists {
		gCtx.Redirect(http.StatusFound, "/")
		return
	}

	userID := user.(gin.H)["id"].(uint)

	// Get user's links
	dbLinks, err := h.paidRouteService.ListUserRoutes(gCtx.Request.Context(), userID)
	if err != nil {
		gCtx.HTML(http.StatusInternalServerError, "dashboard.html", gin.H{
			"error": "Unable to fetch links",
			"user":  user,
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
		"user":    user,
		"links":   uiLinks,
		"host":    host,
		"baseURL": baseURL,
	})
}

// handleSettings handles the settings page
func (h *UIHandler) handleSettings(gCtx *gin.Context) {
	// User is guaranteed to exist due to middleware
	user, exists := gCtx.Get(auth.UserKey)
	if !exists {
		h.logger.Error("auth user not found in context")
		gCtx.Redirect(http.StatusFound, "/")
		return
	}

	// Log the user object for debugging
	h.logger.Info("User from context", "user", fmt.Sprintf("%+v", user))

	userID := user.(gin.H)["id"].(uint)
	h.logger.Info("User ID from context", "userID", userID)

	// Get user details including proxy secret
	userRecord, err := h.userService.GetUserByID(gCtx.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Settings page error: failed to get user details",
			"userID", userID,
			"error", err)
		gCtx.HTML(http.StatusInternalServerError, "settings.html", gin.H{
			"error": "Unable to fetch user details",
			"user":  user,
		})
		return
	}

	// Pass data to the template
	gCtx.HTML(http.StatusOK, "settings.html", gin.H{
		"user":            user,
		"proxy_secret":    userRecord.Proxy402Secret,
		"payment_address": userRecord.PaymentAddress,
	})
}

// handleRegenerateSecret regenerates the user's proxy secret
func (h *UIHandler) handleRegenerateSecret(gCtx *gin.Context) {
	// User is guaranteed to exist due to middleware
	user, exists := gCtx.Get(auth.UserKey)
	if !exists {
		gCtx.Redirect(http.StatusFound, "/")
		return
	}

	userID := user.(gin.H)["id"].(uint)

	// Generate and update the secret
	newSecret, err := h.userService.UpdateProxySecret(gCtx.Request.Context(), userID)
	if err != nil {
		gCtx.HTML(http.StatusInternalServerError, "settings.html", gin.H{
			"error": "Failed to regenerate secret",
			"user":  user,
		})
		return
	}

	// Get full user record to include all fields
	userRecord, _ := h.userService.GetUserByID(gCtx.Request.Context(), userID)
	paymentAddress := ""
	if userRecord != nil {
		paymentAddress = userRecord.PaymentAddress
	}

	// Return form with success message
	gCtx.HTML(http.StatusOK, "settings.html", gin.H{
		"user":            user,
		"proxy_secret":    newSecret,
		"payment_address": paymentAddress,
		"message":         "Secret regenerated successfully",
	})
}

// handleUpdatePaymentAddress handles the update payment address form submission
func (h *UIHandler) handleUpdatePaymentAddress(gCtx *gin.Context) {
	// User is guaranteed to exist due to middleware
	user, exists := gCtx.Get(auth.UserKey)
	if !exists {
		gCtx.Redirect(http.StatusFound, "/")
		return
	}

	userID := user.(gin.H)["id"].(uint)
	paymentAddress := gCtx.PostForm("payment_address")

	// Update payment address
	err := h.userService.UpdatePaymentAddress(gCtx.Request.Context(), userID, paymentAddress)
	if err != nil {
		// Get user record for rendering the form again with error
		userRecord, userErr := h.userService.GetUserByID(gCtx.Request.Context(), userID)
		if userErr != nil {
			userRecord = &users.User{} // empty record if can't fetch
		}

		gCtx.HTML(http.StatusBadRequest, "settings.html", gin.H{
			"error":           "Failed to update payment address: " + err.Error(),
			"user":            user,
			"proxy_secret":    userRecord.Proxy402Secret,
			"payment_address": paymentAddress, // Return the invalid input
		})
		return
	}

	// Return form with success message
	// Get the updated user record
	userRecord, _ := h.userService.GetUserByID(gCtx.Request.Context(), userID)
	proxySecret := ""
	if userRecord != nil {
		proxySecret = userRecord.Proxy402Secret
	}

	// Render updated form
	gCtx.HTML(http.StatusOK, "settings.html", gin.H{
		"user":            user,
		"proxy_secret":    proxySecret,
		"payment_address": paymentAddress,
		"message":         "Payment address updated successfully",
	})
}
