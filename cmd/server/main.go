package main

import (
	"log"

	x402gin "github.com/coinbase/x402/pkg/x402/gin" // Comment out original import
	"github.com/gin-gonic/gin"

	"linkshrink/internal/api/handlers"
	"linkshrink/internal/api/middleware"
	"linkshrink/internal/config"
	"linkshrink/internal/core/services"
	"linkshrink/internal/store"
)

func main() {
	// Load configuration
	config.LoadConfig()

	// Initialize database
	store.InitDatabase()
	db := store.DB // Get the GORM DB instance

	// Create stores
	userStore := store.NewUserStore(db)
	linkStore := store.NewLinkStore(db)

	// Create services
	userService := services.NewUserService(userStore)
	linkService := services.NewLinkService(linkStore)

	// Create handlers
	userHandler := handlers.NewUserHandler(userService)
	linkHandler := handlers.NewLinkHandler(linkService)

	// Setup Gin router
	router := gin.Default() // Includes Logger and Recovery middleware

	// Public routes
	router.POST("/register", userHandler.Register)
	router.POST("/login", userHandler.Login)
	router.GET("/:shortCode", linkHandler.RedirectLink) // Matches any path at the root

	// Group routes that require authentication
	authRequired := router.Group("/")
	authRequired.Use(middleware.AuthMiddleware())
	{

		authRequired.POST("/shrink",
			x402gin.PaymentMiddleware( // Assuming PaymentMiddleware is here
				config.AppConfig.X402Price,
				config.AppConfig.X402PaymentAddress,
				x402gin.WithFacilitatorURL(config.AppConfig.X402FacilitatorURL),
				x402gin.WithResource(config.AppConfig.X402ResourceURL),
			),
			linkHandler.CreateLink, // Then the actual handler
		)

		// User-specific link management
		linksGroup := authRequired.Group("/links")
		{
			linksGroup.GET("", linkHandler.GetUserLinks)          // Get all links for the user
			linksGroup.DELETE("/:linkID", linkHandler.DeleteLink) // Delete a specific link
		}
	}

	// Start server
	appPort := ":" + config.AppConfig.AppPort
	log.Printf("Starting server on port %s", appPort)
	if err := router.Run(appPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
