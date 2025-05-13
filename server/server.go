package server

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"linkshrink/auth"
	"linkshrink/cloudflare"
	"linkshrink/config"
	"linkshrink/purchases"
	"linkshrink/routes"
	"linkshrink/ui"
	"linkshrink/users"
)

// Server represents the HTTP server and its dependencies
type Server struct {
	router          *gin.Engine
	logger          *slog.Logger
	config          *config.Config
	templatesFS     embed.FS
	staticFS        embed.FS
	userService     *users.UserService
	routeService    *routes.PaidRouteService
	purchaseService *purchases.PurchaseService
	authService     *auth.Service
}

// NewServer creates and configures a new server instance
func NewServer(
	userService *users.UserService,
	routeService *routes.PaidRouteService,
	purchaseService *purchases.PurchaseService,
	authService *auth.Service,
	cloudflareService *cloudflare.Service,

	templatesFS embed.FS,
	staticFS embed.FS,

	logger *slog.Logger,
	cfg *config.Config,
) *Server {
	router := gin.Default() // Includes Logger and Recovery middleware

	return &Server{
		router:          router,
		logger:          logger,
		config:          cfg,
		templatesFS:     templatesFS,
		staticFS:        staticFS,
		userService:     userService,
		routeService:    routeService,
		purchaseService: purchaseService,
		authService:     authService,
	}
}

// SetupRoutes configures all application routes
func (s *Server) SetupRoutes() error {
	// Create handlers
	oauthHandler := auth.NewAuthHandler(s.userService, s.authService, &s.config.Auth)
	paidRouteHandler := routes.NewPaidRouteHandler(s.routeService,
		s.purchaseService, s.userService, &s.config.Routes, s.logger)
	uiHandler := ui.NewUIHandler(s.routeService, s.authService, s.userService, &s.config.UI, s.templatesFS, s.logger)
	purchaseHandler := purchases.NewPurchaseHandler(s.purchaseService)

	// Parse HTML templates and set them before registering any routes or it will cause a warning
	tmpl, err := template.ParseFS(s.templatesFS, "templates/*.html")
	if err != nil {
		s.logger.Error("Failed to parse HTML templates", "error", err)
		return err
	}
	s.router.SetHTMLTemplate(tmpl)

	// Serve static files from embedded filesystem
	staticFileSystem, err := fs.Sub(s.staticFS, "static")
	if err != nil {
		log.Fatalf("Failed to create sub filesystem for static files: %v", err)
		return err
	}
	s.router.StaticFS("/static", http.FS(staticFileSystem))

	// Health endpoint
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Service is healthy"})
	})

	// Set up UI routes
	uiHandler.SetupRoutes(s.router)

	// Public routes

	// --- Paid Route Proxy ---
	// This route handles all methods for the dynamic short codes
	s.router.Any("/:shortCode", paidRouteHandler.HandlePaidRoute)

	// Group routes that require authentication
	authRequired := s.router.Group("/")
	authRequired.Use(auth.AuthMiddleware(s.authService)) // Using backward compatibility
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

		// Dashboard data endpoint
		authRequired.GET("/dashboard/stats", purchaseHandler.GetDashboardStats)
	}

	// Logout route
	s.router.GET("/logout", func(c *gin.Context) {
		c.SetCookie("jwt", "", -1, "/", "", false, true)
		c.Redirect(http.StatusFound, "/")
	})
	// OAuth routes
	s.router.GET("/auth/login", oauthHandler.Login)
	s.router.GET("/auth/callback", oauthHandler.Callback)

	return nil
}

// Run starts the HTTP server
func (s *Server) Run() error {
	appPort := ":" + s.config.AppPort
	s.logger.Info("Starting server", "port", appPort)
	return s.router.Run(appPort)
}
