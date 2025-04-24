package services

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"gorm.io/gorm"

	"linkshrink/internal/core/models"
	"linkshrink/internal/core/shortener"
	"linkshrink/internal/store"
)

const maxShortCodeGenerationRetries = 5

// LinkService provides business logic for URL shortening and redirection.
type LinkService struct {
	linkStore *store.LinkStore
}

// NewLinkService creates a new LinkService.
func NewLinkService(linkStore *store.LinkStore) *LinkService {
	return &LinkService{linkStore: linkStore}
}

// CreateShortLink generates a unique short code for a given URL and saves the link.
func (s *LinkService) CreateShortLink(originalURL string, userID uint, expiresAt *time.Time) (*models.Link, error) {
	// Basic URL validation
	parsedURL, err := url.ParseRequestURI(originalURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, errors.New("invalid URL provided")
	}

	var shortCode string
	var link *models.Link
	var creationErr error

	// Try generating a unique short code a few times
	for i := 0; i < maxShortCodeGenerationRetries; i++ {
		shortCode, err = shortener.GenerateSecureShortCode(shortener.DefaultLength)
		if err != nil {
			// If even secure generation fails, something is wrong
			return nil, fmt.Errorf("failed to generate short code: %w", err)
		}

		// Check if the generated code already exists
		_, err = s.linkStore.FindByShortCode(shortCode)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Code is unique, proceed to create
			link = &models.Link{
				OriginalURL: originalURL,
				ShortCode:   shortCode,
				UserID:      userID,
				ExpiresAt:   expiresAt,
			}
			creationErr = s.linkStore.Create(link)
			if creationErr == nil {
				return link, nil // Successfully created
			} else {
				// Handle potential race condition or other DB error during creation
				return nil, fmt.Errorf("failed to save link: %w", creationErr)
			}
		} else if err != nil {
			// An unexpected error occurred during lookup
			return nil, fmt.Errorf("error checking short code uniqueness: %w", err)
		}
		// If we reach here, a collision occurred, loop again to generate a new code
	}

	// If we exhausted retries, return an error
	return nil, errors.New("failed to generate a unique short code after multiple attempts")
}

// GetOriginalURL retrieves the original URL for a short code and increments its visit count.
func (s *LinkService) GetOriginalURL(shortCode string) (string, error) {
	link, err := s.linkStore.FindByShortCode(shortCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("short link not found")
		}
		return "", fmt.Errorf("error retrieving link: %w", err)
	}

	// Check for expiration
	if link.ExpiresAt != nil && time.Now().After(*link.ExpiresAt) {
		// Optionally, delete expired link here or have a separate cleanup process
		return "", errors.New("link has expired")
	}

	// Increment visit count (best effort, ignore error for redirection purposes)
	_ = s.linkStore.IncrementVisitCount(shortCode)

	return link.OriginalURL, nil
}

// GetUserLinks retrieves all links for a specific user.
func (s *LinkService) GetUserLinks(userID uint) ([]models.Link, error) {
	return s.linkStore.FindByUserID(userID)
}

// DeleteLink deletes a link if owned by the user.
func (s *LinkService) DeleteLink(linkID uint, userID uint) error {
	err := s.linkStore.Delete(linkID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Provide a more specific error message
			return errors.New("link not found or you do not have permission to delete it")
		}
		return fmt.Errorf("error deleting link: %w", err)
	}
	return nil
}
