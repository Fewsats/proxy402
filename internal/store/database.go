package store

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"linkshrink/internal/core/models"
)

var DB *gorm.DB

// InitDatabase initializes the database connection and performs migrations.
func InitDatabase() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		getEnv("DB_HOST", "db"),
		getEnv("DB_USER", "user"),
		getEnv("DB_PASSWORD", "password"),
		getEnv("DB_NAME", "linkshrink"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_SSLMODE", "disable"),
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			logger.Config{
				SlowThreshold:             time.Second, // Slow SQL threshold
				LogLevel:                  logger.Info, // Log level
				IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
				Colorful:                  true,        // Disable color
			},
		),
	})

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Database connection established.")

	// --- BACKFILL CODE START ---
	// Purpose: Populate NULL proxy_402_secret for existing users.
	// This should run AFTER the column is made nullable and BEFORE it's made non-nullable again.
	log.Println("Attempting backfill for proxy_402_secret...")

	// We assume the column exists and is nullable at this point.
	type UserIDResult struct {
		ID uint
	}
	var userIDs []UserIDResult
	// Query ALL users (including soft-deleted) with NULL secret
	result := DB.Raw("SELECT id FROM users WHERE proxy_402_secret IS NULL").Scan(&userIDs)

	if result.Error != nil {
		// Log error, but don't necessarily stop. Maybe none exist.
		log.Printf("Warning: Error querying for users with NULL secrets: %v", result.Error)
	}

	if len(userIDs) > 0 {
		log.Printf("Found %d users (incl. soft-deleted) with NULL proxy_402_secret to update.", len(userIDs))
		updatedCount := 0
		errorCount := 0

		tx := DB.Begin()
		if tx.Error != nil {
			log.Fatalf("FATAL: Failed to begin transaction for backfill: %v", tx.Error)
			// No point continuing if transaction fails
		} else {
			for _, uid := range userIDs {
				secretBytes := make([]byte, 16)
				if _, err := rand.Read(secretBytes); err != nil {
					log.Printf("ERROR: Failed to generate secret for user %d: %v. Skipping.", uid.ID, err)
					errorCount++
					continue
				}
				proxySecret := hex.EncodeToString(secretBytes)

				updateResult := tx.Exec("UPDATE users SET proxy_402_secret = ? WHERE id = ?", proxySecret, uid.ID)

				if updateResult.Error != nil {
					log.Printf("ERROR: Failed to update secret for user %d: %v", uid.ID, updateResult.Error)
					errorCount++
				} else if updateResult.RowsAffected > 0 {
					updatedCount++
				} else {
					log.Printf("Warning: No rows affected when updating user %d.", uid.ID)
				}
			}

			if errorCount > 0 {
				log.Printf("ERROR: %d errors occurred during backfill. Rolling back transaction.", errorCount)
				tx.Rollback()
				// Consider if this should be fatal or just a warning
				log.Printf("WARNING: Backfill incomplete due to errors.")
			} else {
				log.Printf("Backfill updates prepared for %d users. Committing transaction.", updatedCount)
				if err := tx.Commit().Error; err != nil {
					log.Printf("ERROR: Failed to commit backfill transaction: %v", err)
					// Consider if this should be fatal
					log.Printf("WARNING: Backfill commit failed.")
				} else {
					log.Println("Backfill transaction committed successfully.")
				}
			}
		}
	} else {
		log.Println("No users found with NULL proxy_402_secret.")
	}
	// --- BACKFILL CODE END ---

	// Run migrations (should be no-op for proxy_402_secret nullability at this stage)
	log.Println("Running AutoMigrate after backfill attempt...")
	err = DB.AutoMigrate(
		&models.User{},
		&models.PaidRoute{},
		&models.Purchase{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("Database migrated successfully.")
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
