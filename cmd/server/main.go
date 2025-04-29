package main

import (
	"log"

	// x402gin "github.com/coinbase/x402/go/pkg/gin" // Comment out original import
	"github.com/gin-gonic/gin"

	"linkshrink/internal/api/handlers"
	"linkshrink/internal/api/middleware"
	"linkshrink/internal/config"
	"linkshrink/internal/core/services"
	"linkshrink/internal/store"
	// No longer importing x402 directly here
)

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
	userHandler := handlers.NewUserHandler(userService)
	paidRouteHandler := handlers.NewPaidRouteHandler(paidRouteService) // Add PaidRouteHandler

	// Setup Gin router
	router := gin.Default() // Includes Logger and Recovery middleware

	// Public routes
	router.POST("/register", userHandler.Register)
	router.POST("/login", userHandler.Login)

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

	// Start server
	appPort := ":" + config.AppConfig.AppPort
	log.Printf("Starting server on port %s", appPort)
	if err := router.Run(appPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
