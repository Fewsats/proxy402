package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"linkshrink/internal/core/models"
	pkgPurchases "linkshrink/purchases"
	"linkshrink/store/sqlc"
)

// Create inserts a new purchase in the database and returns the ID.
func (s *Store) CreatePurchase(ctx context.Context, purchase *models.Purchase) (uint64, error) {
	params := sqlc.CreatePurchaseParams{
		ShortCode:      purchase.ShortCode,
		TargetUrl:      purchase.TargetURL,
		Method:         purchase.Method,
		Price:          int32(purchase.Price),
		IsTest:         purchase.IsTest,
		PaymentPayload: []byte(purchase.PaymentPayload),
		SettleResponse: []byte(purchase.SettleResponse),
		PaidRouteID:    int32(purchase.PaidRouteID),
	}

	ID, err := s.queries.CreatePurchase(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("failed to create purchase: %w", err)
	}

	return uint64(ID), nil
}

// ListByUserID retrieves all purchases for a specific user.
func (s *Store) ListPurchasesByUserID(ctx context.Context, userID uint) ([]models.Purchase, error) {
	dbPurchases, err := s.queries.ListPurchasesByUserID(ctx, int32(userID))
	if err != nil {
		return nil, err
	}

	purchases := make([]models.Purchase, len(dbPurchases))
	for i, dbPurchase := range dbPurchases {
		purchases[i] = *convertToPurchaseModel(dbPurchase)
	}

	return purchases, nil
}

// ListByShortCode retrieves all purchases for a specific shortcode.
func (s *Store) ListPurchasesByShortCode(ctx context.Context, shortCode string) ([]models.Purchase, error) {
	// Using existing query to fetch purchases directly by shortcode
	rows, err := s.db.Query(ctx,
		"SELECT * FROM purchases WHERE short_code = $1 ORDER BY created_at DESC",
		shortCode)
	if err != nil {
		return nil, fmt.Errorf("failed to query purchases by shortcode: %w", err)
	}
	defer rows.Close()

	var purchases []models.Purchase
	for rows.Next() {
		var p sqlc.Purchase
		if err := rows.Scan(
			&p.ID, &p.ShortCode, &p.TargetUrl, &p.Method, &p.Price,
			&p.IsTest, &p.PaymentPayload, &p.SettleResponse, &p.PaidRouteID,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan purchase row: %w", err)
		}
		purchases = append(purchases, *convertToPurchaseModel(p))
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over purchase rows: %w", err)
	}

	if len(purchases) == 0 {
		return nil, pkgPurchases.ErrPurchaseNotFound
	}

	return purchases, nil
}

// GetDailyStatsByUserID retrieves daily purchase stats for a user
func (s *Store) GetDailyStatsByUserID(ctx context.Context, userID uint, days int) ([]pkgPurchases.DailyStats, int64, int, error) {
	// Create pgtype.Text for days parameter
	daysText := pgtype.Text{
		String: strconv.Itoa(days),
		Valid:  true,
	}

	dbStats, err := s.queries.GetDailyStats(ctx, sqlc.GetDailyStatsParams{
		UserID:  int32(userID),
		Column2: daysText,
	})
	if err != nil {
		return nil, 0, 0, err
	}

	// Get total stats
	totalStats, err := s.queries.GetTotalStats(ctx, int32(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []pkgPurchases.DailyStats{}, 0, 0, pkgPurchases.ErrNoStats
		}
		return nil, 0, 0, err
	}

	stats := make([]pkgPurchases.DailyStats, len(dbStats))
	for i, dbStat := range dbStats {
		// Converting TestEarnings and RealEarnings from interface{} to int64
		var testEarnings, realEarnings int64

		if te, ok := dbStat.TestEarnings.(int64); ok {
			testEarnings = te
		}

		if re, ok := dbStat.RealEarnings.(int64); ok {
			realEarnings = re
		}

		stats[i] = pkgPurchases.DailyStats{
			Date:         dbStat.Date,
			Count:        int(dbStat.Count),
			Earnings:     int64(dbStat.Earnings), // Convert int32 to int64
			TestCount:    int(dbStat.TestCount),
			TestEarnings: testEarnings,
			RealCount:    int(dbStat.RealCount),
			RealEarnings: realEarnings,
		}
	}

	// If we need to pad with empty days
	if len(stats) < days {
		stats = padDailyStats(stats, days)
	}

	var totalEarnings int64
	if te, ok := totalStats.TotalEarnings.(int64); ok {
		totalEarnings = te
	}

	return stats, totalEarnings, int(totalStats.TotalCount), nil
}

// padDailyStats fills in missing days with zero values
func padDailyStats(stats []pkgPurchases.DailyStats, days int) []pkgPurchases.DailyStats {
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
			stats = append(stats, pkgPurchases.DailyStats{
				Date:         date,
				Count:        0,
				Earnings:     0,
				TestCount:    0,
				TestEarnings: 0,
				RealCount:    0,
				RealEarnings: 0,
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

	return stats
}

// Helper function to convert sqlc Purchase to models.Purchase
func convertToPurchaseModel(dbPurchase sqlc.Purchase) *models.Purchase {
	purchase := &models.Purchase{
		ShortCode:      dbPurchase.ShortCode,
		TargetURL:      dbPurchase.TargetUrl,
		Method:         dbPurchase.Method,
		Price:          int64(dbPurchase.Price),
		IsTest:         dbPurchase.IsTest,
		PaymentPayload: string(dbPurchase.PaymentPayload),
		SettleResponse: string(dbPurchase.SettleResponse),
		PaidRouteID:    uint(dbPurchase.PaidRouteID),
		CreatedAt:      dbPurchase.CreatedAt,
		UpdatedAt:      dbPurchase.UpdatedAt,
	}

	return purchase
}
