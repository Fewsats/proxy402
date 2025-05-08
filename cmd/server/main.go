package main

import (
	"embed"
	"log/slog"

	betterstackPkg "github.com/samber/slog-betterstack"

	"linkshrink/auth"
	"linkshrink/config"
	"linkshrink/purchases"
	"linkshrink/routes"
	"linkshrink/server"
	storePkg "linkshrink/store"
	"linkshrink/users"
	"linkshrink/utils"
)

//go:embed templates
var templatesFS embed.FS

//go:embed static
var staticFS embed.FS

// configureLogger creates a logger based on configuration
func configureLogger(cfg *config.Config) *slog.Logger {
	// Start with default logger
	logger := slog.Default()

	// Configure BetterStack if credentials are available
	if cfg.BetterStack.Token != "" && cfg.BetterStack.Endpoint != "" {
		logger = slog.New(
			betterstackPkg.Option{
				Token:    cfg.BetterStack.Token,
				Endpoint: cfg.BetterStack.Endpoint,
			}.NewBetterstackHandler(),
		)
		logger.Info("BetterStack logging enabled")
	}

	return logger
}

func main() {
	// Use the default logger until the configuration is loaded.
	logger := slog.Default()

	// Load the configuration.
	cfg := config.LoadConfig(logger)

	// Configure the logger based on settings
	logger = configureLogger(cfg)

	logger.Info(
		"Logger configuration",
		"level", cfg.LogLevel,
	)

	if err := cfg.SetLoggerLevel(); err != nil {
		logger.Error(
			"Unable to set logger level",
			"error", err,
		)
		return
	}

	cfg.SetGinMode()

	// Initialize the store.
	clock := utils.NewRealClock()
	store, err := storePkg.NewStore(logger, &cfg.Store, clock)
	if err != nil {
		logger.Error(
			"Unable to create store",
			"error", err,
		)

		return
	}

	userService := users.NewUserService(logger, store)
	paidRouteService := routes.NewPaidRouteService(logger, store)
	purchaseService := purchases.NewPurchaseService(logger, store)
	authService := auth.NewAuthService(&cfg.Auth)

	// Create and configure the server
	srv := server.NewServer(
		logger,
		cfg,
		userService,
		paidRouteService,
		purchaseService,
		authService,
		templatesFS,
		staticFS,
	)

	// Setup routes
	if err := srv.SetupRoutes(); err != nil {
		logger.Error(
			"Failed to set up routes",
			"error", err,
		)
		return
	}

	// Start the server
	if err := srv.Run(); err != nil {
		logger.Error(
			"Server failed to start",
			"error", err,
		)
		return
	}
}
