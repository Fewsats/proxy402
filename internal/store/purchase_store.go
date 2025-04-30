package store

import (
	"linkshrink/internal/core/models"
	"time"

	"gorm.io/gorm"
)

// PurchaseStore defines methods for interacting with purchase data.
type PurchaseStore struct {
	db *gorm.DB
}

// NewPurchaseStore creates a new PurchaseStore.
func NewPurchaseStore(db *gorm.DB) *PurchaseStore {
	return &PurchaseStore{db: db}
}

// Create inserts a new purchase record into the database.
func (s *PurchaseStore) Create(purchase *models.Purchase) error {
	result := s.db.Create(purchase)
	return result.Error
}

// ListByShortCode retrieves all purchases for a specific short code.
func (s *PurchaseStore) ListByShortCode(shortCode string) ([]models.Purchase, error) {
	var purchases []models.Purchase
	result := s.db.Where("short_code = ?", shortCode).Order("created_at desc").Find(&purchases)
	return purchases, result.Error
}

// ListByUserID retrieves all purchases for a specific user.
// This joins with the PaidRoute table to filter by UserID.
func (s *PurchaseStore) ListByUserID(userID uint) ([]models.Purchase, error) {
	var purchases []models.Purchase
	result := s.db.Joins("JOIN paid_routes ON purchases.paid_route_id = paid_routes.id").
		Where("paid_routes.user_id = ?", userID).
		Order("purchases.created_at desc").
		Find(&purchases)
	return purchases, result.Error
}

// DailyStats represents purchase statistics for a single day
type DailyStats struct {
	Date     string `json:"date"`
	Count    int    `json:"count"`
	Earnings int64  `json:"earnings"`
}

// GetDailyStatsByUserID retrieves daily purchase stats for a user
func (s *PurchaseStore) GetDailyStatsByUserID(userID uint, days int) ([]DailyStats, int64, int, error) {
	var stats []DailyStats
	var totalEarnings int64
	var totalCount int

	// First get total earnings with COALESCE to handle NULL case
	type TotalStats struct {
		Total int64
		Count int
	}
	var totals TotalStats

	result := s.db.Model(&models.Purchase{}).
		Joins("JOIN paid_routes ON purchases.paid_route_id = paid_routes.id").
		Where("paid_routes.user_id = ?", userID).
		Select("COALESCE(SUM(purchases.price), 0) as total, COUNT(*) as count").
		Scan(&totals)
	if result.Error != nil {
		return nil, 0, 0, result.Error
	}

	totalEarnings = totals.Total
	totalCount = totals.Count

	// Get daily stats grouped by date
	result = s.db.Model(&models.Purchase{}).
		Joins("JOIN paid_routes ON purchases.paid_route_id = paid_routes.id").
		Where("paid_routes.user_id = ?", userID).
		Select("DATE(purchases.created_at) as date, COUNT(*) as count, SUM(purchases.price) as earnings").
		Group("DATE(purchases.created_at)").
		Order("date desc").
		Scan(&stats)

	// If no data or need to fill more days, generate empty days
	if len(stats) == 0 || len(stats) < days {
		today := time.Now().UTC()
		existingDates := make(map[string]bool)

		// Mark existing dates
		for _, s := range stats {
			existingDates[s.Date] = true
		}

		// Fill missing dates
		for i := 0; i < days; i++ {
			date := today.AddDate(0, 0, -i).Format("2006-01-02")
			if !existingDates[date] {
				stats = append(stats, DailyStats{
					Date:     date,
					Count:    0,
					Earnings: 0,
				})
			}
		}

		// Resort by date desc
		for i := 0; i < len(stats)-1; i++ {
			for j := 0; j < len(stats)-i-1; j++ {
				if stats[j].Date < stats[j+1].Date {
					stats[j], stats[j+1] = stats[j+1], stats[j]
				}
			}
		}

		// Limit to requested days
		if len(stats) > days {
			stats = stats[:days]
		}
	}

	return stats, totalEarnings, totalCount, result.Error
}
