package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"linkshrink/internal/api/handlers"
	"linkshrink/internal/api/middleware"
	"linkshrink/internal/config"
	"linkshrink/internal/core/models"
	"linkshrink/internal/core/services"
	"linkshrink/internal/store"
	// No longer importing x402 directly here
)

//go:embed templates
var templatesFS embed.FS

func main() {
	// Load configuration
	config.LoadConfig()

	// Initialize database
	store.InitDatabase()
	db := store.DB // Get the GORM DB instance

	// Create stores
	userStore := store.NewUserStore(db)
	paidRouteStore := store.NewPaidRouteStore(db) // Add PaidRouteStore

	// Create services
	userService := services.NewUserService(userStore)
	paidRouteService := services.NewPaidRouteService(paidRouteStore) // Add PaidRouteService

	// Create handlers
	// userHandler := handlers.NewUserHandler(userService)
	oauthHandler := handlers.NewOAuthHandler(userService)

	paidRouteHandler := handlers.NewPaidRouteHandler(paidRouteService) // Add PaidRouteHandler

	// Setup Gin router
	router := gin.Default() // Includes Logger and Recovery middleware

	// Public routes
	// TODO probably remove these

	// --- Paid Route Proxy ---
	// This route handles all methods for the dynamic short codes
	router.Any("/:shortCode", paidRouteHandler.HandlePaidRoute)

	// Group routes that require authentication
	authRequired := router.Group("/")
	authRequired.Use(middleware.AuthMiddleware())
	{
		// Original /shrink endpoint for simple link shortening (kept for now?)
		// Consider if this is still needed or if everything should be a paid route.
		// Rerouting /links/shrink to create a PaidRoute instead of a standard Link
		authRequired.POST("/links/shrink", paidRouteHandler.CreatePaidRouteHandler)

		// User-specific link management (standard links)
		// These might become obsolete if only PaidRoutes are used
		// Renaming group to /routes might be clearer, but keeping /links for now.
		linksGroup := authRequired.Group("/links") // Or rename to "/routes"?
		{
			linksGroup.GET("", paidRouteHandler.GetUserPaidRoutes)
			linksGroup.DELETE("/:linkID", paidRouteHandler.DeleteUserPaidRoute) // Note: Param is still :linkID
		}
	}

	// Parse HTML templates
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatalf("failed to parse HTML templates: %v", err)
	}
	router.SetHTMLTemplate(tmpl)

	// Main UI route
	router.GET("/", middleware.AuthMiddleware(), func(c *gin.Context) {
		// Check if user is authenticated
		user, exists := c.Get(middleware.UserKey)

		var links []models.PaidRoute
		var err error

		if exists {
			// Get user's links if authenticated
			userID := user.(gin.H)["id"].(uint)
			links, err = paidRouteService.ListUserRoutes(userID)
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
			"links":   links,
			"host":    host,
			"baseURL": baseURL,
		})
	})

	// Logout route
	router.GET("/logout", func(c *gin.Context) {
		c.SetCookie("jwt", "", -1, "/", "", false, true)
		c.Redirect(http.StatusFound, "/")
	})
	// OAuth routes
	router.GET("/auth/login", oauthHandler.Login)
	router.GET("/auth/callback", oauthHandler.Callback)

	// Start server
	appPort := ":" + config.AppConfig.AppPort
	log.Printf("Starting server on port %s", appPort)
	if err := router.Run(appPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
