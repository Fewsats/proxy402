package services

import (
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strings"

	"linkshrink/internal/core/models"
	"linkshrink/internal/core/shortener"
	"linkshrink/internal/store"
)

// PaidRouteService provides business logic for managing paid routes.
type PaidRouteService struct {
	routeStore *store.PaidRouteStore
}

// NewPaidRouteService creates a new PaidRouteService.
func NewPaidRouteService(routeStore *store.PaidRouteStore) *PaidRouteService {
	return &PaidRouteService{routeStore: routeStore}
}

var validMethods = map[string]bool{
	"GET":    true,
	"POST":   true,
	"PUT":    true,
	"DELETE": true,
	"PATCH":  true,
}

// CreatePaidRoute validates input, generates a unique short code, and saves the route.
func (s *PaidRouteService) CreatePaidRoute(targetURL, method, priceStr string, userID uint) (*models.PaidRoute, error) {
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

	// 3. Validate Price String
	priceFloat, ok := new(big.Float).SetString(priceStr)
	if !ok {
		return nil, errors.New("invalid price format provided")
	}
	if priceFloat.Sign() < 0 { // Check if price is negative
		return nil, errors.New("price cannot be negative")
	}
	// We store the original validated string `priceStr`

	// 4. Generate Unique Short Code
	const maxShortCodeGenerationRetries = 10
	var shortCode string
	for i := 0; i < maxShortCodeGenerationRetries; i++ {
		shortCode, err = shortener.GenerateSecureShortCode(shortener.DefaultLength)
		if err != nil {
			return nil, fmt.Errorf("failed to generate short code: %w", err)
		}

		exists, err := s.routeStore.CheckShortCodeExists(shortCode)
		if err != nil {
			return nil, fmt.Errorf("error checking short code uniqueness: %w", err)
		}
		if !exists {
			break // Found unique code
		}

		if i == maxShortCodeGenerationRetries-1 {
			return nil, errors.New("failed to generate a unique short code after multiple attempts")
		}
	}

	// 5. Create and Save Route
	route := &models.PaidRoute{
		ShortCode: shortCode,
		TargetURL: targetURL,
		Method:    upperMethod,
		Price:     priceStr, // Store the original validated string
		UserID:    userID,
		IsEnabled: true, // Default to enabled
	}

	if err := s.routeStore.Create(route); err != nil {
		// Handle potential race condition on unique index
		return nil, fmt.Errorf("failed to save paid route: %w", err)
	}

	return route, nil
}

// FindEnabledRouteByShortCode retrieves an active route.
func (s *PaidRouteService) FindEnabledRouteByShortCode(shortCode string) (*models.PaidRoute, error) {
	route, err := s.routeStore.FindByShortCode(shortCode)
	if err != nil {
		if errors.Is(err, errors.New("record not found")) { // GORM might return different error types
			return nil, errors.New("route not found or not enabled")
		}
		return nil, fmt.Errorf("error retrieving route: %w", err)
	}
	return route, nil
}

// IncrementPaymentCount attempts to increment the count for a given short code.
// It ignores not found errors, as the route might have been disabled/deleted.
func (s *PaidRouteService) IncrementPaymentCount(shortCode string) {
	// Best effort: ignore error if record not found or already disabled
	err := s.routeStore.IncrementPaymentCount(shortCode)
	if err != nil && !errors.Is(err, errors.New("record not found")) { // Check for gorm.ErrRecordNotFound maybe?
		// Log other errors if needed
		fmt.Printf("Error incrementing payment count for %s: %v\n", shortCode, err)
	}
}

// ListUserRoutes retrieves all paid routes for a specific user.
func (s *PaidRouteService) ListUserRoutes(userID uint) ([]models.PaidRoute, error) {
	return s.routeStore.ListByUserID(userID)
}

// DeleteRoute deletes a paid route if owned by the specified user.
func (s *PaidRouteService) DeleteRoute(routeID uint, userID uint) error {
	err := s.routeStore.Delete(routeID, userID)
	if err != nil {
		// Check if the error is gorm.ErrRecordNotFound
		if errors.Is(err, errors.New("record not found")) { // Adjust error check as needed
			return errors.New("route not found or you do not have permission to delete it")
		}
		return fmt.Errorf("error deleting route: %w", err)
	}
	return nil
}

// TODO: Add services for Update etc. later.
