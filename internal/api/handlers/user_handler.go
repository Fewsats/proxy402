package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"linkshrink/internal/core/services"
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
func (h *UserHandler) GetProfile(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	userIDUint, ok := userID.(uint)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.userService.GetUserByID(userIDUint)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user profile"})
		return
	}

	// Return user profile without sensitive information
	userResponse := gin.H{
		"id":        user.ID,
		"email":     user.Email,
		"createdAt": user.CreatedAt,
	}

	ctx.JSON(http.StatusOK, userResponse)
}
