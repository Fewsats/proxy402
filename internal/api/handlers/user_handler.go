package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"linkshrink/internal/core/services"
	"linkshrink/users"
)

// UserHandler handles HTTP requests related to users.
type UserHandler struct {
	userService *services.UserService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// GetProfile returns the current user's profile information
func (h *UserHandler) GetProfile(gCtx *gin.Context) {
	userID, exists := gCtx.Get("userID")
	if !exists {
		gCtx.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		gCtx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.userService.GetUserByID(gCtx.Request.Context(), userIDUint)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			gCtx.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		gCtx.Error(err)
		gCtx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user profile"})
		return
	}

	// Return user profile without sensitive information
	userResponse := gin.H{
		"id":        user.ID,
		"email":     user.Email,
		"createdAt": user.CreatedAt,
	}

	gCtx.JSON(http.StatusOK, userResponse)
}
