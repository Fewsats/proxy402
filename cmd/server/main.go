package main

import (
	"embed"
	"log/slog"

	"linkshrink/config"
	"linkshrink/server"
	storePkg "linkshrink/store"
	"linkshrink/utils"
)

//go:embed templates
var templatesFS embed.FS

//go:embed static
var staticFS embed.FS

func main() {
	// Use the default logger until the configuration is loaded.
	logger := slog.Default()

	// Load the configuration.
	cfg := config.LoadConfig(logger)

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

	// Create services
	userService := services.NewUserService(logger, store)
	paidRouteService := services.NewPaidRouteService(logger, store)
	purchaseService := services.NewPurchaseService(logger, store)

	// Create and configure the server
	srv := server.NewServer(
		logger,
		cfg,
		userService,
		paidRouteService,
		purchaseService,
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
