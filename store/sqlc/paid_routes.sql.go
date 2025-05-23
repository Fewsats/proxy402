// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: paid_routes.sql

package sqlc

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const checkShortCodeExists = `-- name: CheckShortCodeExists :one
SELECT EXISTS(
  SELECT 1 FROM paid_routes
  WHERE short_code = $1
) as exists
`

// CheckShortCodeExists checks if a short code already exists.
func (q *Queries) CheckShortCodeExists(ctx context.Context, shortCode string) (bool, error) {
	row := q.db.QueryRow(ctx, checkShortCodeExists, shortCode)
	var exists bool
	err := row.Scan(&exists)
	return exists, err
}

const createPaidRoute = `-- name: CreatePaidRoute :one
INSERT INTO paid_routes (
    short_code, target_url, method, price, is_test,
    user_id, is_enabled, attempt_count, payment_count, access_count,
    created_at, updated_at,
    type, credits, resource_type, original_filename, cover_url,
    title, description
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
    $13, $14, $15, $16, $17, $18, $19
) RETURNING id, short_code, target_url, method, price, is_test, user_id, is_enabled, attempt_count, payment_count, access_count, created_at, updated_at, deleted_at, type, credits, resource_type, original_filename, cover_url, title, description
`

type CreatePaidRouteParams struct {
	ShortCode        string
	TargetUrl        string
	Method           string
	Price            int32
	IsTest           bool
	UserID           int64
	IsEnabled        bool
	AttemptCount     int32
	PaymentCount     int32
	AccessCount      int32
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Type             string
	Credits          int32
	ResourceType     string
	OriginalFilename pgtype.Text
	CoverUrl         pgtype.Text
	Title            pgtype.Text
	Description      pgtype.Text
}

// CreatePaidRoute creates a new paid route.
func (q *Queries) CreatePaidRoute(ctx context.Context, arg CreatePaidRouteParams) (PaidRoute, error) {
	row := q.db.QueryRow(ctx, createPaidRoute,
		arg.ShortCode,
		arg.TargetUrl,
		arg.Method,
		arg.Price,
		arg.IsTest,
		arg.UserID,
		arg.IsEnabled,
		arg.AttemptCount,
		arg.PaymentCount,
		arg.AccessCount,
		arg.CreatedAt,
		arg.UpdatedAt,
		arg.Type,
		arg.Credits,
		arg.ResourceType,
		arg.OriginalFilename,
		arg.CoverUrl,
		arg.Title,
		arg.Description,
	)
	var i PaidRoute
	err := row.Scan(
		&i.ID,
		&i.ShortCode,
		&i.TargetUrl,
		&i.Method,
		&i.Price,
		&i.IsTest,
		&i.UserID,
		&i.IsEnabled,
		&i.AttemptCount,
		&i.PaymentCount,
		&i.AccessCount,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
		&i.Type,
		&i.Credits,
		&i.ResourceType,
		&i.OriginalFilename,
		&i.CoverUrl,
		&i.Title,
		&i.Description,
	)
	return i, err
}

const deletePaidRoute = `-- name: DeletePaidRoute :exec
UPDATE paid_routes SET
    deleted_at = $3
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
`

type DeletePaidRouteParams struct {
	ID        int64
	UserID    int64
	DeletedAt pgtype.Timestamptz
}

// DeletePaidRoute soft-deletes a paid route.
func (q *Queries) DeletePaidRoute(ctx context.Context, arg DeletePaidRouteParams) error {
	_, err := q.db.Exec(ctx, deletePaidRoute, arg.ID, arg.UserID, arg.DeletedAt)
	return err
}

const getEnabledPaidRouteByShortCode = `-- name: GetEnabledPaidRouteByShortCode :one
SELECT id, short_code, target_url, method, price, is_test, user_id, is_enabled, attempt_count, payment_count, access_count, created_at, updated_at, deleted_at, type, credits, resource_type, original_filename, cover_url, title, description FROM paid_routes
WHERE short_code = $1 AND is_enabled = true AND deleted_at IS NULL
`

// GetEnabledPaidRouteByShortCode returns an enabled paid route by its short code.
func (q *Queries) GetEnabledPaidRouteByShortCode(ctx context.Context, shortCode string) (PaidRoute, error) {
	row := q.db.QueryRow(ctx, getEnabledPaidRouteByShortCode, shortCode)
	var i PaidRoute
	err := row.Scan(
		&i.ID,
		&i.ShortCode,
		&i.TargetUrl,
		&i.Method,
		&i.Price,
		&i.IsTest,
		&i.UserID,
		&i.IsEnabled,
		&i.AttemptCount,
		&i.PaymentCount,
		&i.AccessCount,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
		&i.Type,
		&i.Credits,
		&i.ResourceType,
		&i.OriginalFilename,
		&i.CoverUrl,
		&i.Title,
		&i.Description,
	)
	return i, err
}

const getPaidRouteByID = `-- name: GetPaidRouteByID :one
SELECT id, short_code, target_url, method, price, is_test, user_id, is_enabled, attempt_count, payment_count, access_count, created_at, updated_at, deleted_at, type, credits, resource_type, original_filename, cover_url, title, description FROM paid_routes
WHERE id = $1 AND deleted_at IS NULL
`

// GetPaidRouteByID returns a paid route by ID.
func (q *Queries) GetPaidRouteByID(ctx context.Context, id int64) (PaidRoute, error) {
	row := q.db.QueryRow(ctx, getPaidRouteByID, id)
	var i PaidRoute
	err := row.Scan(
		&i.ID,
		&i.ShortCode,
		&i.TargetUrl,
		&i.Method,
		&i.Price,
		&i.IsTest,
		&i.UserID,
		&i.IsEnabled,
		&i.AttemptCount,
		&i.PaymentCount,
		&i.AccessCount,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
		&i.Type,
		&i.Credits,
		&i.ResourceType,
		&i.OriginalFilename,
		&i.CoverUrl,
		&i.Title,
		&i.Description,
	)
	return i, err
}

const getPaidRouteByShortCode = `-- name: GetPaidRouteByShortCode :one
SELECT id, short_code, target_url, method, price, is_test, user_id, is_enabled, attempt_count, payment_count, access_count, created_at, updated_at, deleted_at, type, credits, resource_type, original_filename, cover_url, title, description FROM paid_routes
WHERE short_code = $1 AND deleted_at IS NULL
`

// GetPaidRouteByShortCode returns a paid route by its short code.
func (q *Queries) GetPaidRouteByShortCode(ctx context.Context, shortCode string) (PaidRoute, error) {
	row := q.db.QueryRow(ctx, getPaidRouteByShortCode, shortCode)
	var i PaidRoute
	err := row.Scan(
		&i.ID,
		&i.ShortCode,
		&i.TargetUrl,
		&i.Method,
		&i.Price,
		&i.IsTest,
		&i.UserID,
		&i.IsEnabled,
		&i.AttemptCount,
		&i.PaymentCount,
		&i.AccessCount,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
		&i.Type,
		&i.Credits,
		&i.ResourceType,
		&i.OriginalFilename,
		&i.CoverUrl,
		&i.Title,
		&i.Description,
	)
	return i, err
}

const incrementAccessCount = `-- name: IncrementAccessCount :exec
UPDATE paid_routes SET
    access_count = access_count + 1,
    updated_at = $2
WHERE short_code = $1 AND deleted_at IS NULL
`

type IncrementAccessCountParams struct {
	ShortCode string
	UpdatedAt time.Time
}

// IncrementAccessCount increments the access_count for a route.
func (q *Queries) IncrementAccessCount(ctx context.Context, arg IncrementAccessCountParams) error {
	_, err := q.db.Exec(ctx, incrementAccessCount, arg.ShortCode, arg.UpdatedAt)
	return err
}

const incrementAttemptCount = `-- name: IncrementAttemptCount :exec
UPDATE paid_routes SET
    attempt_count = attempt_count + 1,
    updated_at = $2
WHERE short_code = $1 AND deleted_at IS NULL
`

type IncrementAttemptCountParams struct {
	ShortCode string
	UpdatedAt time.Time
}

// IncrementAttemptCount increments the attempt_count for a route.
func (q *Queries) IncrementAttemptCount(ctx context.Context, arg IncrementAttemptCountParams) error {
	_, err := q.db.Exec(ctx, incrementAttemptCount, arg.ShortCode, arg.UpdatedAt)
	return err
}

const incrementPaymentCount = `-- name: IncrementPaymentCount :exec
UPDATE paid_routes SET
    payment_count = payment_count + 1,
    updated_at = $2
WHERE short_code = $1 AND deleted_at IS NULL
`

type IncrementPaymentCountParams struct {
	ShortCode string
	UpdatedAt time.Time
}

// IncrementPaymentCount increments the payment_count for a route.
func (q *Queries) IncrementPaymentCount(ctx context.Context, arg IncrementPaymentCountParams) error {
	_, err := q.db.Exec(ctx, incrementPaymentCount, arg.ShortCode, arg.UpdatedAt)
	return err
}

const listUserPaidRoutes = `-- name: ListUserPaidRoutes :many
SELECT id, short_code, target_url, method, price, is_test, user_id, is_enabled, attempt_count, payment_count, access_count, created_at, updated_at, deleted_at, type, credits, resource_type, original_filename, cover_url, title, description FROM paid_routes
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
`

// ListUserPaidRoutes returns all paid routes for a specific user.
func (q *Queries) ListUserPaidRoutes(ctx context.Context, userID int64) ([]PaidRoute, error) {
	rows, err := q.db.Query(ctx, listUserPaidRoutes, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []PaidRoute
	for rows.Next() {
		var i PaidRoute
		if err := rows.Scan(
			&i.ID,
			&i.ShortCode,
			&i.TargetUrl,
			&i.Method,
			&i.Price,
			&i.IsTest,
			&i.UserID,
			&i.IsEnabled,
			&i.AttemptCount,
			&i.PaymentCount,
			&i.AccessCount,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.DeletedAt,
			&i.Type,
			&i.Credits,
			&i.ResourceType,
			&i.OriginalFilename,
			&i.CoverUrl,
			&i.Title,
			&i.Description,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
