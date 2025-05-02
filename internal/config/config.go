package config

import (
	"linkshrink/store"
	"log"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Config holds application configuration values.
type Config struct {
	// Logging configuration.
	LogLevel string `long:"log_level" description:"Logging level {trace, debug, info, warn, error, critical}"`

	// Gin mode.
	GinMode string `long:"gin_mode" description:"Gin mode {debug, release}"`

	// Store configuration.
	Store store.Config `group:"store" namespace:"store"`

	AppPort            string
	JWTSecret          string
	JWTExpirationHours time.Duration

	// x402 Config
	X402TestnetPaymentAddress string `mapstructure:"X402_TESTNET_PAYMENT_ADDRESS"` // Testnet Payment Address
	X402MainnetPaymentAddress string `mapstructure:"X402_MAINNET_PAYMENT_ADDRESS"` // Mainnet Payment Address
	X402FacilitatorURL        string `mapstructure:"X402_FACILITATOR_URL"`
	GoogleClientID            string `mapstructure:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret        string `mapstructure:"GOOGLE_CLIENT_SECRET"`

	// OAuth Config
	GoogleOAuth *oauth2.Config
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

// LoadConfig loads configuration from environment variables or a .env file.
func LoadConfig(logger *slog.Logger) *Config {
	// Load .env file, ignore error if it doesn't exist (e.g., in production)
	_ = godotenv.Load()

	dbPort, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
		log.Fatalf("FATAL: DB_PORT environment variable is not set.")
	}

	AppConfig = &Config{
		// Store configuration.
		Store: store.Config{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     dbPort,
			User:     getEnv("DB_USER", "user"),
			Password: getEnv("DB_PASSWORD", "password"),
			DBName:   getEnv("DB_NAME", "linkshrink"),
		},

		AppPort:            getEnv("APP_PORT", "8080"),
		JWTSecret:          getEnvOrFatal("JWT_SECRET"),
		JWTExpirationHours: getEnvDuration("JWT_EXPIRATION_HOURS", 72*time.Hour),

		// Load x402 Config
		X402TestnetPaymentAddress: getEnvOrFatal("X402_TESTNET_PAYMENT_ADDRESS"),
		X402MainnetPaymentAddress: getEnvOrFatal("X402_MAINNET_PAYMENT_ADDRESS"),
		X402FacilitatorURL:        getEnv("X402_FACILITATOR_URL", "https://x402.org/facilitator"),

		// Load Google OAuth config
		GoogleClientID:     getEnvOrFatal("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: getEnvOrFatal("GOOGLE_CLIENT_SECRET"),

		LogLevel: getEnv("LOG_LEVEL", "debug"),
		GinMode:  getEnv("GIN_MODE", "debug"),
	}

	// Basic validation for essential x402 config
	if AppConfig.X402TestnetPaymentAddress == "" {
		log.Fatal("FATAL: X402_TESTNET_PAYMENT_ADDRESS environment variable is not set.")
	}
	if AppConfig.X402MainnetPaymentAddress == "" {
		log.Fatal("FATAL: X402_MAINNET_PAYMENT_ADDRESS environment variable is not set.")
	}

	// Initialize Google OAuth config
	AppConfig.GoogleOAuth = &oauth2.Config{
		ClientID:     AppConfig.GoogleClientID,
		ClientSecret: AppConfig.GoogleClientSecret,
		RedirectURL:  getEnvOrFatal("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

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
