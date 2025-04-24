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

// RegisterRequest defines the expected JSON body for user registration.
type RegisterRequest struct {
	Username string `json:"username" binding:"required,alphanum,min=3,max=30"`
	Password string `json:"password" binding:"required,min=6"`
}

// Register handles user registration requests.
func (h *UserHandler) Register(ctx *gin.Context) {
	var req RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	user, err := h.userService.Register(req.Username, req.Password)
	if err != nil {
		// Check for specific errors like "username already taken"
		if err.Error() == "username already taken" {
			ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else {
			// Log the internal error for debugging
			ctx.Error(err) // Gin logs this by default if using Logger middleware
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		}
		return
	}

	// Exclude password from response
	userResponse := gin.H{
		"id":        user.ID,
		"username":  user.Username,
		"createdAt": user.CreatedAt,
	}

	ctx.JSON(http.StatusCreated, userResponse)
}

// LoginRequest defines the expected JSON body for user login.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login handles user login requests.
func (h *UserHandler) Login(ctx *gin.Context) {
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	token, err := h.userService.Login(req.Username, req.Password)
	if err != nil {
		if err.Error() == "invalid username or password" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		} else {
			ctx.Error(err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed"})
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"token": token})
}
