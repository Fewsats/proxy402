package routes

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"linkshrink/cloudflare"
	"log/slog"
	"mime/multipart"
	"net/url"
	"strings"
	"time"
)

// PaidRouteService provides business logic for managing paid routes.
type PaidRouteService struct {
	cloudflareService *cloudflare.Service
	priceUtils        PriceUtils

	logger *slog.Logger
	store  Store
}

// NewPaidRouteService creates a new PaidRouteService.
func NewPaidRouteService(logger *slog.Logger, store Store, cloudflareService *cloudflare.Service) *PaidRouteService {
	return &PaidRouteService{
		logger:            logger,
		store:             store,
		cloudflareService: cloudflareService,
		priceUtils:        NewPriceUtils(),
	}
}

var validMethods = map[string]bool{
	"GET":    true,
	"POST":   true,
	"PUT":    true,
	"DELETE": true,
	"PATCH":  true,
}

func generateCoverImageKey(userID uint64) string {
	return fmt.Sprintf("%d/%d", userID, time.Now().UnixNano())
}

// CreateURLRoute validates input, generates a unique short code, and saves the route.
func (s *PaidRouteService) CreateURLRoute(ctx context.Context, req *CreatePaidRouteRequest, userID uint64) (*PaidRoute, error) {

	// 1. Validate target URL
	parsedURL, err := url.ParseRequestURI(req.TargetURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, errors.New("invalid target URL provided")
	}

	// 2. Convert price string to integer (USDC * 10^6)
	priceInt, err := s.priceUtils.ParsePrice(req.Price)
	if err != nil {
		return nil, err
	}

	// 3. Create and Save Route (short code will be generated in the store)
	route := &PaidRoute{
		TargetURL:              req.TargetURL,
		Method:                 strings.ToUpper(req.Method),
		Price:                  priceInt,
		IsTest:                 req.IsTest,
		UserID:                 userID,
		IsEnabled:              true,
		Type:                   req.Type,
		Credits:                req.Credits,
		PaymentProtocolVersion: PaymentProtocolVersionV2,
		ResourceType:           "url",
	}

	// Handle title and description if provided
	if req.Title != "" {
		route.Title = &req.Title
	}

	if req.Description != "" {
		route.Description = &req.Description
	}

	// Handle cover image if provided
	if req.CoverImage != nil {
		coverURL, err := s.ProcessAndUploadCoverImage(ctx,
			generateCoverImageKey(userID), req.CoverImage)
		if err != nil {
			return nil, fmt.Errorf("failed to upload cover image: %w", err)
		}
		route.CoverImageURL = coverURL
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

// CreateFileRoute creates a new paid route for a file and returns the route along with a signed upload URL.
func (s *PaidRouteService) CreateFileRoute(ctx context.Context, req *CreateFileRouteRequest, userID uint64) (*PaidRoute, string, error) {

	// Convert price string to integer (USDC * 10^6)
	priceInt, err := s.priceUtils.ParsePrice(req.Price)
	if err != nil {
		return nil, "", err
	}

	// Generate unique R2 key using userID and timestamp
	r2Key := fmt.Sprintf("%d/%d", userID, time.Now().UnixNano())
	coverImageKey := generateCoverImageKey(userID)

	// Generate signed upload URL
	uploadURL, err := s.cloudflareService.GetUploadURL(ctx, r2Key)
	if err != nil {
		// Consider deleting the route if we can't get an upload URL
		return nil, "", fmt.Errorf("failed to generate upload URL: %w", err)
	}

	// Create a new route with resource_type = "file"
	route := &PaidRoute{
		TargetURL:              r2Key,
		Method:                 "GET", // Always use GET for file downloads
		Price:                  priceInt,
		IsTest:                 req.IsTest,
		UserID:                 userID,
		IsEnabled:              true,
		Type:                   req.Type,
		Credits:                req.Credits,
		PaymentProtocolVersion: PaymentProtocolVersionV2,
		ResourceType:           "file",
		OriginalFilename:       &req.OriginalFilename,
	}

	// Handle title and description if provided
	if req.Title != "" {
		route.Title = &req.Title
	}

	if req.Description != "" {
		route.Description = &req.Description
	}

	// Handle cover image if provided
	if req.CoverImage != nil {
		coverURL, err := s.ProcessAndUploadCoverImage(ctx,
			coverImageKey, req.CoverImage)
		if err != nil {
			s.logger.Error("Failed to upload cover image", "error", err)
			return nil, "", fmt.Errorf("failed to upload cover image: %w", err)
		}
		route.CoverImageURL = coverURL
	}

	// Save the route
	createdRoute, err := s.store.CreateRoute(ctx, route)
	if err != nil {
		return nil, "", fmt.Errorf("failed to save file route: %w", err)
	}

	return createdRoute, uploadURL, nil
}

// GetFileDownloadURL generates a presigned URL for downloading a file from R2
func (s *PaidRouteService) GetFileDownloadURL(ctx context.Context, key string, originalFilename string) (string, error) {
	// Use the cloudflare service to generate a presigned download URL
	downloadURL, err := s.cloudflareService.GetDownloadURL(ctx, key, originalFilename)
	if err != nil {
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}

	return downloadURL, nil
}

// ProcessAndUploadCoverImage processes and uploads a cover image to R2
func (s *PaidRouteService) ProcessAndUploadCoverImage(gCtx context.Context,
	externalID string, coverImageHeader *multipart.FileHeader) (*string, error) {

	file, err := coverImageHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open cover image: %w", err)
	}
	defer file.Close()

	coverImageBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read cover image: %w", err)
	}

	coverImageReader := bytes.NewReader(coverImageBytes)

	coverURL, err := s.cloudflareService.UploadPublicFile(gCtx, externalID,
		"cover-images", coverImageReader)
	if err != nil {
		return nil, fmt.Errorf("failed to upload cover file: %w", err)
	}

	return &coverURL, nil
}
