package handlers

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"linkshrink/internal/api/middleware"
	"linkshrink/internal/core/models"
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

// SetupRoutes registers UI routes to the provided router
func (h *UIHandler) SetupRoutes(router *gin.Engine) {
	// Parse HTML templates
	tmpl, err := template.ParseFS(h.templatesFS, "templates/*.html")
	if err != nil {
		log.Fatalf("failed to parse HTML templates: %v", err)
	}
	router.SetHTMLTemplate(tmpl)

	router.GET("/", middleware.AuthMiddleware(), h.handleIndex)
}

// handleIndex handles the main index page
func (h *UIHandler) handleIndex(c *gin.Context) {
	// Check if user is authenticated
	user, exists := c.Get(middleware.UserKey)

	var dbLinks []models.PaidRoute
	var uiLinks []UIPaidRoute
	var err error

	if exists {
		// Get user's links if authenticated
		userID := user.(gin.H)["id"].(uint)
		dbLinks, err = h.paidRouteService.ListUserRoutes(userID)

		// Convert DB models to UI models
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

		if err != nil {
			c.HTML(http.StatusInternalServerError, "main.html", gin.H{
				"error": "Unable to fetch links",
			})
			return
		}
	}

	// Get the scheme and host for generating full URLs
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	host := c.Request.Host
	baseURL := scheme + "://" + host

	c.HTML(http.StatusOK, "main.html", gin.H{
		"user":    user,
		"links":   uiLinks,
		"host":    host,
		"baseURL": baseURL,
	})
}
