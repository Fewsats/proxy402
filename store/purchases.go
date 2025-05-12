package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	pkgPurchases "linkshrink/purchases"
	"linkshrink/store/sqlc"
)

// Create inserts a new purchase in the database and returns the ID.
func (s *Store) CreatePurchase(ctx context.Context, purchase *pkgPurchases.Purchase) (uint64, error) {
	now := s.clock.Now()
	params := sqlc.CreatePurchaseParams{
		ShortCode:        purchase.ShortCode,
		TargetUrl:        purchase.TargetURL,
		Method:           purchase.Method,
		Price:            int32(purchase.Price),
		Type:             purchase.Type,
		CreditsAvailable: int32(purchase.CreditsAvailable),
		CreditsUsed:      int32(purchase.CreditsUsed),
		IsTest:           purchase.IsTest,

		PaidRouteID:   int64(purchase.PaidRouteID),
		PaidToAddress: purchase.PaidToAddress,

		PaymentHeader:  pgtype.Text{String: purchase.PaymentHeader, Valid: purchase.PaymentHeader != ""},
		PaymentPayload: []byte(purchase.PaymentPayload),
		SettleResponse: []byte(purchase.SettleResponse),

		CreatedAt: now,
		UpdatedAt: now,
	}

	ID, err := s.queries.CreatePurchase(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("failed to create purchase: %w", err)
	}

	return uint64(ID), nil
}

// ListByUserID retrieves all purchases for a specific user.
func (s *Store) ListPurchasesByUserID(ctx context.Context,
	userID uint64) ([]pkgPurchases.Purchase, error) {

	dbPurchases, err := s.queries.ListPurchasesByUserID(ctx, int64(userID))
	if err != nil {
		return nil, err
	}

	purchases := make([]pkgPurchases.Purchase, len(dbPurchases))
	for i, dbPurchase := range dbPurchases {
		purchases[i] = *convertToPurchaseModel(dbPurchase)
	}

	return purchases, nil
}

// ListByShortCode retrieves all purchases for a specific shortcode.
func (s *Store) ListPurchasesByShortCode(ctx context.Context, shortCode string) ([]pkgPurchases.Purchase, error) {
	// Using existing query to fetch purchases directly by shortcode
	rows, err := s.db.Query(ctx,
		"SELECT * FROM purchases WHERE short_code = $1 ORDER BY created_at DESC",
		shortCode)
	if err != nil {
		return nil, fmt.Errorf("failed to query purchases by shortcode: %w", err)
	}
	defer rows.Close()

	var purchases []pkgPurchases.Purchase
	for rows.Next() {
		var p sqlc.Purchase
		if err := rows.Scan(
			&p.ID, &p.ShortCode, &p.TargetUrl, &p.Method, &p.Price,
			&p.IsTest, &p.PaymentPayload, &p.SettleResponse, &p.PaidRouteID,
			&p.PaidToAddress, &p.CreatedAt, &p.UpdatedAt,
			&p.Type, &p.CreditsAvailable, &p.CreditsUsed, &p.PaymentHeader,
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
func (s *Store) GetDailyStatsByUserID(ctx context.Context,
	userID uint64, days uint64) ([]pkgPurchases.DailyStats, error) {

	// Create pgtype.Text for days parameter
	daysText := pgtype.Text{
		String: strconv.Itoa(int(days)),
		Valid:  true,
	}

	dbStats, err := s.queries.GetDailyStats(ctx, sqlc.GetDailyStatsParams{
		UserID:  int64(userID),
		Column2: daysText,
	})
	if err != nil {
		return nil, err
	}

	// // Get total stats
	// totalStats, err := s.queries.GetTotalStats(ctx, int64(userID))
	// if err != nil {
	// 	if errors.Is(err, pgx.ErrNoRows) {
	// 		return []pkgPurchases.DailyStats{}, pkgPurchases.ErrNoStats
	// 	}
	// 	return nil, err
	// }

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
			Count:        uint64(dbStat.Count),
			Earnings:     uint64(dbStat.Earnings), // Convert int32 to int64
			TestCount:    uint64(dbStat.TestCount),
			TestEarnings: uint64(testEarnings),
			RealCount:    uint64(dbStat.RealCount),
			RealEarnings: uint64(realEarnings),
		}
	}

	// If we need to pad with empty days
	if len(stats) < int(days) {
		stats = padDailyStats(stats, days)
	}

	// var totalEarnings uint64
	// if te, ok := totalStats.TotalEarnings.(int64); ok {
	// 	totalEarnings = uint64(te)
	// }

	return stats, nil
}

// padDailyStats fills in missing days with zero values
func padDailyStats(stats []pkgPurchases.DailyStats, days uint64) []pkgPurchases.DailyStats {
	today := time.Now().UTC()
	existingDates := make(map[string]bool)

	// Mark existing dates
	for _, s := range stats {
		existingDates[s.Date] = true
	}

	// Fill missing dates
	for i := 0; i < int(days); i++ {
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
	if len(stats) > int(days) {
		stats = stats[:int(days)]
	}

	return stats
}

// Helper function to convert sqlc Purchase to pkgPurchases.Purchase
func convertToPurchaseModel(dbPurchase sqlc.Purchase) *pkgPurchases.Purchase {
	var paymentHeaderStr string
	if dbPurchase.PaymentHeader.Valid {
		paymentHeaderStr = dbPurchase.PaymentHeader.String
	}

	purchase := &pkgPurchases.Purchase{
		ID:               uint64(dbPurchase.ID),
		ShortCode:        dbPurchase.ShortCode,
		TargetURL:        dbPurchase.TargetUrl,
		Method:           dbPurchase.Method,
		Price:            uint64(dbPurchase.Price),
		Type:             dbPurchase.Type,
		CreditsAvailable: uint64(dbPurchase.CreditsAvailable),
		CreditsUsed:      uint64(dbPurchase.CreditsUsed),
		IsTest:           dbPurchase.IsTest,

		PaidRouteID:   uint64(dbPurchase.PaidRouteID),
		PaidToAddress: dbPurchase.PaidToAddress,

		PaymentHeader:  paymentHeaderStr,
		PaymentPayload: []byte(dbPurchase.PaymentPayload),
		SettleResponse: []byte(dbPurchase.SettleResponse),

		CreatedAt: dbPurchase.CreatedAt,
		UpdatedAt: dbPurchase.UpdatedAt,
	}

	return purchase
}

func (s *Store) GetPurchaseByRouteIDAndPaymentHeader(ctx context.Context, routeID uint64, paymentHeader string) (*pkgPurchases.Purchase, error) {
	params := sqlc.GetPurchaseByRouteIDAndPaymentHeaderParams{
		PaidRouteID:   int64(routeID),
		PaymentHeader: pgtype.Text{String: paymentHeader, Valid: paymentHeader != ""},
	}
	dbPurchase, err := s.queries.GetPurchaseByRouteIDAndPaymentHeader(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pkgPurchases.ErrPurchaseNotFound
		}
		return nil, fmt.Errorf("failed to get purchase by route ID and payment header: %w", err)
	}
	return convertToPurchaseModel(dbPurchase), nil
}

// IncrementPurchaseCreditsUsed increments the credits_used count for a specific purchase ID.
func (s *Store) IncrementPurchaseCreditsUsed(ctx context.Context, purchaseID uint64) error {
	params := sqlc.IncrementPurchaseCreditsUsedParams{
		ID:        int64(purchaseID),
		UpdatedAt: s.clock.Now(),
	}
	err := s.queries.IncrementPurchaseCreditsUsed(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to execute increment credits_used query: %w", err)
	}
	return nil
}
