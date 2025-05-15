package config

import (
	"log"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"linkshrink/auth"
	"linkshrink/cloudflare"
	"linkshrink/routes"
	"linkshrink/store"
	"linkshrink/ui"
)

// Config holds application configuration values.
type Config struct {
	// Logging configuration.
	LogLevel string `long:"log_level" description:"Logging level {trace, debug, info, warn, error, critical}"`

	// Gin mode.
	GinMode string `long:"gin_mode" description:"Gin mode {debug, release}"`

	// Store configuration.
	Store store.Config `group:"store" namespace:"store"`

	// Routes configuration
	Routes routes.Config `group:"routes" namespace:"routes"`

	AppPort string

	// Auth configuration
	Auth auth.Config `group:"auth" namespace:"auth"`

	// BetterStack logging configuration
	BetterStack struct {
		Token    string
		Endpoint string
	}

	// UI configuration
	UI ui.Config `group:"ui" namespace:"ui"`

	// Cloudflare configuration
	Cloudflare cloudflare.Config `group:"cloudflare" namespace:"cloudflare"`
}

var AppConfig *Config

// SetLoggerLevel sets the logger level based on the configuration.
func (c *Config) SetLoggerLevel() error {
	switch c.LogLevel {
	case "info":
		slog.SetLogLoggerLevel(slog.LevelInfo)
	case "debug":
		slog.SetLogLoggerLevel(slog.LevelDebug)
	case "warn":
		slog.SetLogLoggerLevel(slog.LevelWarn)
	case "error":
		slog.SetLogLoggerLevel(slog.LevelError)
	}
	return nil
}

// SetGinMode sets the gin mode based on the configuration.
func (c *Config) SetGinMode() {
	if c.GinMode != "" {
		gin.SetMode(c.GinMode)
	}
}

// DefaultConfig returns default values for the Config struct.
func DefaultConfig() *Config {
	return &Config{
		LogLevel: "debug",
		GinMode:  "debug",
		AppPort:  "8080",
		Store: store.Config{
			Host:     "localhost",
			Port:     5432,
			User:     "user",
			Password: "password",
			DBName:   "linkshrink",
		},
		Routes: routes.Config{
			X402FacilitatorURL: "https://x402.org/facilitator",
		},
		Auth: auth.Config{
			JWTExpirationHours: 72 * time.Hour,
		},
		UI:         ui.DefaultConfig(),
		Cloudflare: cloudflare.DefaultConfig(),
	}
}

// LoadConfig loads configuration from environment variables or a .env file.
func LoadConfig(logger *slog.Logger) *Config {
	// Start with default values
	AppConfig = DefaultConfig()

	// Load .env file, ignore error if it doesn't exist (e.g., in production)
	_ = godotenv.Load()

	dbPort, err := strconv.Atoi(getEnv("DB_PORT", strconv.Itoa(AppConfig.Store.Port)))
	if err != nil {
		log.Fatalf("FATAL: DB_PORT environment variable is not valid.")
	}

	// Override defaults with environment variables
	AppConfig.LogLevel = getEnv("LOG_LEVEL", AppConfig.LogLevel)
	AppConfig.GinMode = getEnv("GIN_MODE", AppConfig.GinMode)
	AppConfig.AppPort = getEnv("APP_PORT", AppConfig.AppPort)

	// Store configuration.
	AppConfig.Store.Host = getEnv("DB_HOST", AppConfig.Store.Host)
	AppConfig.Store.Port = dbPort
	AppConfig.Store.User = getEnv("DB_USER", AppConfig.Store.User)
	AppConfig.Store.Password = getEnv("DB_PASSWORD", AppConfig.Store.Password)
	AppConfig.Store.DBName = getEnv("DB_NAME", AppConfig.Store.DBName)

	// Check if migrations should be skipped
	skipMigrations, _ := strconv.ParseBool(getEnv("DB_SKIP_MIGRATIONS", "false"))
	AppConfig.Store.SkipMigrations = skipMigrations

	// Routes configuration
	AppConfig.Routes.X402PaymentAddress = getEnvOrFatal("X402_PAYMENT_ADDRESS")
	AppConfig.Routes.X402FacilitatorURL = getEnv("X402_FACILITATOR_URL", AppConfig.Routes.X402FacilitatorURL)
	AppConfig.Routes.CDPAPIKeyID = getEnv("CDP_API_KEY_ID", "")
	AppConfig.Routes.CDPAPIKeySecret = getEnv("CDP_API_KEY_SECRET", "")

	// Auth configuration
	AppConfig.Auth.JWTSecret = getEnvOrFatal("JWT_SECRET")
	AppConfig.Auth.JWTExpirationHours = getEnvDuration("JWT_EXPIRATION_HOURS", AppConfig.Auth.JWTExpirationHours)

	// Load auth config
	AppConfig.Auth.GoogleClientID = getEnvOrFatal("GOOGLE_CLIENT_ID")
	AppConfig.Auth.GoogleClientSecret = getEnvOrFatal("GOOGLE_CLIENT_SECRET")
	AppConfig.Auth.GoogleRedirectURL = getEnvOrFatal("GOOGLE_REDIRECT_URL")

	// BetterStack configuration
	AppConfig.BetterStack.Token = getEnv("BETTERSTACK_TOKEN", "")
	AppConfig.BetterStack.Endpoint = getEnv("BETTERSTACK_ENDPOINT", "")

	// UI configuration
	AppConfig.UI.GoogleAnalyticsID = getEnv("GOOGLE_ANALYTICS_ID", AppConfig.UI.GoogleAnalyticsID)

	// Cloudflare configuration
	AppConfig.Cloudflare.Endpoint = getEnv("CLOUDFLARE_R2_ENDPOINT", "")
	AppConfig.Cloudflare.AccessKey = getEnv("CLOUDFLARE_R2_ACCESS_KEY", "")
	AppConfig.Cloudflare.SecretAccessKey = getEnv("CLOUDFLARE_R2_SECRET_ACCESS_KEY", "")
	AppConfig.Cloudflare.BucketName = getEnv("CLOUDFLARE_R2_BUCKET_NAME", "")

	logger.Info("Configuration loaded.")

	return AppConfig
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// getEnvOrFatal retrieves an environment variable or logs a fatal error if not found.
func getEnvOrFatal(key string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Fatalf("FATAL: Environment variable %s is not set.", key)
	return "" // Unreachable, but satisfies compiler
}

// getEnvDuration retrieves an environment variable as a time.Duration or returns a default.
func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if valueStr, exists := os.LookupEnv(key); exists {
		if valueInt, err := strconv.Atoi(valueStr); err == nil {
			return time.Duration(valueInt) * time.Hour
		}
		log.Printf("Warning: Invalid format for %s environment variable. Using default: %v", key, fallback)
	}
	return fallback
}
