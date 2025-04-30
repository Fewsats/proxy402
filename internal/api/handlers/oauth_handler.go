package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"linkshrink/internal/auth"
	"linkshrink/internal/config"
	"linkshrink/internal/core/services"
)

// OAuthHandler handles OAuth authentication requests
type OAuthHandler struct {
	userService *services.UserService
}

// NewOAuthHandler creates a new OAuthHandler
func NewOAuthHandler(userService *services.UserService) *OAuthHandler {
	return &OAuthHandler{
		userService: userService,
	}
}

// Login initiates Google OAuth flow
func (h *OAuthHandler) Login(c *gin.Context) {
	// Check if user already has a valid JWT
	cookie, err := c.Cookie("jwt")
	if err == nil && cookie != "" {
		// Already authenticated, redirect to dashboard
		c.Redirect(http.StatusFound, "/dashboard")
		return
	}

	// Not authenticated, proceed with OAuth flow
	url := config.AppConfig.GoogleOAuth.AuthCodeURL("state-token")
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// Callback handles the Google OAuth callback
func (h *OAuthHandler) Callback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.HTML(http.StatusBadRequest, "landing.html", gin.H{
			"error":   "Missing code parameter",
			"baseURL": c.Request.Host,
		})
		return
	}

	token, err := config.AppConfig.GoogleOAuth.Exchange(c.Request.Context(), code)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "landing.html", gin.H{
			"error":   "Failed to exchange token",
			"baseURL": c.Request.Host,
		})
		return
	}

	client := config.AppConfig.GoogleOAuth.Client(c.Request.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		c.HTML(http.StatusInternalServerError, "landing.html", gin.H{
			"error":   "Failed to get user info",
			"baseURL": c.Request.Host,
		})
		return
	}
	defer resp.Body.Close()

	userData, err := io.ReadAll(resp.Body)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "landing.html", gin.H{
			"error":   "Failed to read user data",
			"baseURL": c.Request.Host,
		})
		return
	}

	var userInfo struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		ID    string `json:"id"`
	}
	if err := json.Unmarshal(userData, &userInfo); err != nil {
		c.HTML(http.StatusInternalServerError, "landing.html", gin.H{
			"error":   "Failed to parse user data",
			"baseURL": c.Request.Host,
		})
		return
	}

	// Find or create user
	user, err := h.userService.FindOrCreateUser(userInfo.Email, userInfo.Name, userInfo.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "landing.html", gin.H{
			"error":   "Failed to create or find user",
			"baseURL": c.Request.Host,
		})
		return
	}

	// Generate JWT token
	signedToken, err := auth.GenerateJWT(user.ID, user.Email)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "landing.html", gin.H{
			"error":   "Failed to generate token",
			"baseURL": c.Request.Host,
		})
		return
	}

	// Set the JWT as a secure cookie
	maxAge := int(config.AppConfig.JWTExpirationHours.Seconds())
	c.SetCookie("jwt", signedToken, maxAge, "/", "", false, true)

	// Redirect to dashboard instead of home page
	c.Redirect(http.StatusFound, "/dashboard")
}
