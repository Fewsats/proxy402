package config

import (
	"log"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds application configuration values.
type Config struct {
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	DBSslMode          string
	AppPort            string
	JWTSecret          string
	JWTExpirationHours time.Duration

	// x402 Config
	X402Price          *big.Float
	X402PaymentAddress string
	X402FacilitatorURL string
	X402ResourceURL    string
}

var AppConfig *Config

// LoadConfig loads configuration from environment variables or a .env file.
func LoadConfig() {
	// Load .env file, ignore error if it doesn't exist (e.g., in production)
	_ = godotenv.Load()

	AppConfig = &Config{
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnv("DB_PORT", "5432"),
		DBUser:             getEnv("DB_USER", "user"),
		DBPassword:         getEnv("DB_PASSWORD", "password"),
		DBName:             getEnv("DB_NAME", "linkshrink"),
		DBSslMode:          getEnv("DB_SSLMODE", "disable"),
		AppPort:            getEnv("APP_PORT", "8080"),
		JWTSecret:          getEnvOrFatal("JWT_SECRET"),
		JWTExpirationHours: getEnvDuration("JWT_EXPIRATION_HOURS", 72*time.Hour),

		// Load x402 Config
		X402Price:          getEnvBigFloatOrFatal("X402_PRICE"),
		X402PaymentAddress: getEnvOrFatal("X402_PAYMENT_ADDRESS"),
		X402FacilitatorURL: getEnv("X402_FACILITATOR_URL", "https://x402.org/facilitator"), // Default facilitator
		X402ResourceURL:    getEnv("X402_RESOURCE_URL", ""),                                // Optional, middleware might detect
	}

	// Basic validation for essential x402 config
	if AppConfig.X402PaymentAddress == "" {
		log.Fatal("FATAL: X402_PAYMENT_ADDRESS environment variable is not set.")
	}

	log.Println("Configuration loaded.")
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

// getEnvBigFloatOrFatal retrieves an environment variable as a big.Float or logs fatal error.
func getEnvBigFloatOrFatal(key string) *big.Float {
	valueStr := getEnvOrFatal(key) // Ensures the variable exists
	valueFloat, ok := new(big.Float).SetString(valueStr)
	if !ok {
		log.Fatalf("FATAL: Invalid format for %s environment variable. Expected float string, got: %s", key, valueStr)
	}
	return valueFloat
}
