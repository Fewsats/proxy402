package services

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
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
func (s *PaidRouteService) CreatePaidRoute(targetURL, method, priceStr string, isTest bool, userID uint) (*models.PaidRoute, error) {
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
	priceInt := int64(priceFloat * 1000000)

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
		Price:     priceInt, // Store as int64
		IsTest:    isTest,   // Save the test flag
		UserID:    userID,
		IsEnabled: true,
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

// IncrementPaymentCount increments the payment count for a given short code.
func (s *PaidRouteService) IncrementPaymentCount(shortCode string) error {
	// Delegate to the store layer
	err := s.routeStore.IncrementPaymentCount(shortCode)
	if err != nil {
		// Handle specific errors like not found if needed, otherwise wrap
		if errors.Is(err, store.ErrRouteNotFound) { // Assuming store defines ErrRouteNotFound
			return fmt.Errorf("cannot increment payment count, route %s not found: %w", shortCode, err)
		}
		return fmt.Errorf("failed to increment payment count for %s: %w", shortCode, err)
	}
	return nil
}

// IncrementAttemptCount increments the attempt count for a given short code.
func (s *PaidRouteService) IncrementAttemptCount(shortCode string) error {
	// Delegate to the store layer
	err := s.routeStore.IncrementAttemptCount(shortCode)
	if err != nil {
		if errors.Is(err, store.ErrRouteNotFound) { // Assuming store defines ErrRouteNotFound
			return fmt.Errorf("cannot increment attempt count, route %s not found: %w", shortCode, err)
		}
		return fmt.Errorf("failed to increment attempt count for %s: %w", shortCode, err)
	}
	return nil
}

// IncrementAccessCount increments the access count for a given short code.
func (s *PaidRouteService) IncrementAccessCount(shortCode string) error {
	// Delegate to the store layer
	err := s.routeStore.IncrementAccessCount(shortCode)
	if err != nil {
		if errors.Is(err, store.ErrRouteNotFound) { // Assuming store defines ErrRouteNotFound
			return fmt.Errorf("cannot increment access count, route %s not found: %w", shortCode, err)
		}
		return fmt.Errorf("failed to increment access count for %s: %w", shortCode, err)
	}
	return nil
}

// ListUserRoutes retrieves all routes associated with a specific user ID.
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
