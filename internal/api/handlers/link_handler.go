package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"linkshrink/internal/api/middleware"
	"linkshrink/internal/auth"
	"linkshrink/internal/core/models" // Added for Link response formatting
	"linkshrink/internal/core/services"
)

// LinkHandler handles HTTP requests related to links.
type LinkHandler struct {
	linkService *services.LinkService
}

// NewLinkHandler creates a new LinkHandler.
func NewLinkHandler(linkService *services.LinkService) *LinkHandler {
	return &LinkHandler{linkService: linkService}
}

// CreateLinkRequest defines the expected JSON body for creating a link.
type CreateLinkRequest struct {
	OriginalURL string `json:"original_url" binding:"required,url"`
	// Optional: Allow users to specify expiration, e.g., in hours from now
	ExpiresInHours *int `json:"expires_in_hours,omitempty"`
}

// CreateLink handles requests to create a new short link.
// Requires authentication.
func (h *LinkHandler) CreateLink(ctx *gin.Context) {
	var req CreateLinkRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Get user ID from the context (set by AuthMiddleware)
	authPayload, exists := ctx.Get(middleware.AuthorizationPayloadKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"}) // Should technically be caught by middleware
		return
	}
	payload := authPayload.(*auth.Claims)

	var expiresAt *time.Time
	if req.ExpiresInHours != nil && *req.ExpiresInHours > 0 {
		exp := time.Now().Add(time.Duration(*req.ExpiresInHours) * time.Hour)
		expiresAt = &exp
	}

	link, err := h.linkService.CreateShortLink(req.OriginalURL, payload.UserID, expiresAt)
	if err != nil {
		if err.Error() == "invalid URL provided" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			ctx.Error(err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create short link"})
		}
		return
	}

	// Construct the full short URL to return
	// Scheme might be http or https depending on deployment
	// Host is the request host
	// We assume http for now, could be made configurable
	shortURL := fmt.Sprintf("http://%s/%s", ctx.Request.Host, link.ShortCode)

	ctx.JSON(http.StatusCreated, gin.H{
		"id":           link.ID,
		"original_url": link.OriginalURL,
		"short_code":   link.ShortCode,
		"short_url":    shortURL, // Provide the full clickable URL
		"created_at":   link.CreatedAt,
		"expires_at":   link.ExpiresAt,
		"visit_count":  link.VisitCount,
	})
}

// RedirectLink handles redirection requests for short codes.
func (h *LinkHandler) RedirectLink(ctx *gin.Context) {
	shortCode := ctx.Param("shortCode")
	if shortCode == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Short code parameter is missing"})
		return
	}

	originalURL, err := h.linkService.GetOriginalURL(shortCode)
	if err != nil {
		if err.Error() == "short link not found" || err.Error() == "link has expired" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			ctx.Error(err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve link"})
		}
		return
	}

	// Perform the redirect
	ctx.Redirect(http.StatusFound, originalURL)
}

// formatLinkResponse formats a models.Link for API responses.
func formatLinkResponse(link models.Link, requestHost string) gin.H {
	shortURL := fmt.Sprintf("http://%s/%s", requestHost, link.ShortCode)
	return gin.H{
		"id":           link.ID,
		"original_url": link.OriginalURL,
		"short_code":   link.ShortCode,
		"short_url":    shortURL,
		"created_at":   link.CreatedAt,
		"expires_at":   link.ExpiresAt,
		"visit_count":  link.VisitCount,
	}
}

// GetUserLinks handles requests to retrieve all links for the authenticated user.
// Requires authentication.
func (h *LinkHandler) GetUserLinks(ctx *gin.Context) {
	// Get user ID from the context
	authPayload, exists := ctx.Get(middleware.AuthorizationPayloadKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	payload := authPayload.(*auth.Claims)

	links, err := h.linkService.GetUserLinks(payload.UserID)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user links"})
		return
	}

	// Format the response
	responseLinks := make([]gin.H, len(links))
	for i, link := range links {
		responseLinks[i] = formatLinkResponse(link, ctx.Request.Host)
	}

	ctx.JSON(http.StatusOK, responseLinks)
}

// DeleteLink handles requests to delete a specific link.
// Requires authentication.
func (h *LinkHandler) DeleteLink(ctx *gin.Context) {
	// Get link ID from path parameter
	linkIDStr := ctx.Param("linkID")
	linkID, err := strconv.ParseUint(linkIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid link ID format"})
		return
	}

	// Get user ID from the context
	authPayload, exists := ctx.Get(middleware.AuthorizationPayloadKey)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	payload := authPayload.(*auth.Claims)

	err = h.linkService.DeleteLink(uint(linkID), payload.UserID)
	if err != nil {
		if err.Error() == "link not found or you do not have permission to delete it" {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			ctx.Error(err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete link"})
		}
		return
	}

	ctx.Status(http.StatusNoContent) // Success, no content to return
}
