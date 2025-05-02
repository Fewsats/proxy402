package handlers

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"linkshrink/auth"
	"linkshrink/internal/core/services"
)

// UIHandler handles UI-related routes and rendering
type UIHandler struct {
	paidRouteService *services.PaidRouteService
	templatesFS      embed.FS
}

// NewUIHandler creates a new UIHandler instance
func NewUIHandler(paidRouteService *services.PaidRouteService, templatesFS embed.FS) *UIHandler {
	return &UIHandler{
		paidRouteService: paidRouteService,
		templatesFS:      templatesFS,
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
	router.GET("/dashboard", auth.AuthMiddleware(), h.handleDashboard)
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
