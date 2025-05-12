package routes

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
)

// PaidRouteService provides business logic for managing paid routes.
type PaidRouteService struct {
	logger *slog.Logger

	store Store
}

// NewPaidRouteService creates a new PaidRouteService.
func NewPaidRouteService(logger *slog.Logger, store Store) *PaidRouteService {
	return &PaidRouteService{logger: logger, store: store}
}

var validMethods = map[string]bool{
	"GET":    true,
	"POST":   true,
	"PUT":    true,
	"DELETE": true,
	"PATCH":  true,
}

// CreatePaidRoute validates input, generates a unique short code, and saves the route.
func (s *PaidRouteService) CreatePaidRoute(ctx context.Context, targetURL,
	method, priceStr string, isTest bool, userID uint64, routeType string, credits uint64) (*PaidRoute, error) {

	// 1. Validate Target URL
	parsedURL, err := url.ParseRequestURI(targetURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, errors.New("invalid target URL provided")
	}

	// 2. Validate Method
	upperMethod := strings.ToUpper(method)
	if !validMethods[upperMethod] {
		return nil, errors.New("invalid HTTP method provided")
	}

	// 3. Parse and Validate Price String (decimal string representing USDC, to be converted to integer * 10^6)
	priceFloat, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return nil, errors.New("invalid price format: must be a decimal number")
	}
	if priceFloat < 0 {
		return nil, errors.New("price must be greater or equal to 0")
	}

	// Convert to integer (USDC * 10^6)
	priceInt := uint64(priceFloat * 1000000)

	// Create and Save Route (short code will be generated in the store)
	route := &PaidRoute{
		TargetURL: targetURL,
		Method:    upperMethod,
		Price:     priceInt, // Store as int64
		IsTest:    isTest,   // Save the test flag
		UserID:    userID,
		IsEnabled: true,
		Type:      routeType,
		Credits:   credits,
	}

	createdRoute, err := s.store.CreateRoute(ctx, route)
	if err != nil {
		// Handle potential race condition on unique index
		return nil, fmt.Errorf("failed to save paid route: %w", err)
	}

	return createdRoute, nil
}

// FindEnabledRouteByShortCode retrieves an active route.
func (s *PaidRouteService) FindEnabledRouteByShortCode(ctx context.Context, shortCode string) (*PaidRoute, error) {
	return s.store.FindEnabledRouteByShortCode(ctx, shortCode)
}

// IncrementPaymentCount increments the payment count for a given short code.
func (s *PaidRouteService) IncrementPaymentCount(ctx context.Context, shortCode string) error {
	// Delegate to the store layer
	err := s.store.IncrementRoutePaymentCount(ctx, shortCode)
	if err != nil {
		return fmt.Errorf("failed to increment payment count for %s: %w", shortCode, err)
	}
	return nil
}

// IncrementAttemptCount increments the attempt count for a given short code.
func (s *PaidRouteService) IncrementAttemptCount(ctx context.Context, shortCode string) error {
	// Delegate to the store layer
	err := s.store.IncrementRouteAttemptCount(ctx, shortCode)
	if err != nil {
		return fmt.Errorf("failed to increment attempt count for %s: %w", shortCode, err)
	}
	return nil
}

// IncrementAccessCount increments the access count for a given short code.
func (s *PaidRouteService) IncrementAccessCount(ctx context.Context, shortCode string) error {
	// Delegate to the store layer
	err := s.store.IncrementRouteAccessCount(ctx, shortCode)
	if err != nil {
		return fmt.Errorf("failed to increment access count for %s: %w", shortCode, err)
	}
	return nil
}

// ListUserRoutes retrieves all routes associated with a specific user ID.
func (s *PaidRouteService) ListUserRoutes(ctx context.Context, userID uint64) ([]PaidRoute, error) {
	return s.store.ListUserRoutes(ctx, userID)
}

// DeleteRoute deletes a paid route if owned by the specified user.
func (s *PaidRouteService) DeleteRoute(ctx context.Context, routeID uint64, userID uint64) error {
	err := s.store.DeleteRoute(ctx, routeID, userID)
	if err != nil {
		if errors.Is(err, ErrRouteNotFound) {
			return ErrRouteNotFound
		} else if errors.Is(err, ErrRouteNoPermission) {
			return ErrRouteNoPermission
		}
		return fmt.Errorf("error deleting route: %w", err)
	}
	return nil
}
