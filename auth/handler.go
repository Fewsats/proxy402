package auth

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"linkshrink/config"
	"linkshrink/users"
)

// OAuthHandler handles OAuth authentication requests
type OAuthHandler struct {
	userService *users.UserService
}

// NewOAuthHandler creates a new OAuthHandler
func NewOAuthHandler(userService *users.UserService) *OAuthHandler {
	return &OAuthHandler{
		userService: userService,
	}
}

// Login initiates Google OAuth flow
func (h *OAuthHandler) Login(gCtx *gin.Context) {
	// Check if user already has a valid JWT
	cookie, err := gCtx.Cookie("jwt")
	if err == nil && cookie != "" {
		// Already authenticated, redirect to dashboard
		gCtx.Redirect(http.StatusFound, "/dashboard")
		return
	}

	// Not authenticated, proceed with OAuth flow
	url := config.AppConfig.GoogleOAuth.AuthCodeURL("state-token")
	gCtx.Redirect(http.StatusTemporaryRedirect, url)
}

// Callback handles the Google OAuth callback
func (h *OAuthHandler) Callback(gCtx *gin.Context) {
	code := gCtx.Query("code")
	if code == "" {
		gCtx.HTML(http.StatusBadRequest, "landing.html", gin.H{
			"error":   "Missing code parameter",
			"baseURL": gCtx.Request.Host,
		})
		return
	}

	token, err := config.AppConfig.GoogleOAuth.Exchange(gCtx.Request.Context(), code)
	if err != nil {
		gCtx.HTML(http.StatusInternalServerError, "landing.html", gin.H{
			"error":   "Failed to exchange token",
			"baseURL": gCtx.Request.Host,
		})
		return
	}

	client := config.AppConfig.GoogleOAuth.Client(gCtx.Request.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		gCtx.HTML(http.StatusInternalServerError, "landing.html", gin.H{
			"error":   "Failed to get user info",
			"baseURL": gCtx.Request.Host,
		})
		return
	}
	defer resp.Body.Close()

	userData, err := io.ReadAll(resp.Body)
	if err != nil {
		gCtx.HTML(http.StatusInternalServerError, "landing.html", gin.H{
			"error":   "Failed to read user data",
			"baseURL": gCtx.Request.Host,
		})
		return
	}

	var userInfo struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		ID    string `json:"id"`
	}
	if err := json.Unmarshal(userData, &userInfo); err != nil {
		gCtx.HTML(http.StatusInternalServerError, "landing.html", gin.H{
			"error":   "Failed to parse user data",
			"baseURL": gCtx.Request.Host,
		})
		return
	}

	// Find or create user
	user, err := h.userService.FindOrCreateUser(gCtx.Request.Context(), userInfo.Email, userInfo.Name, userInfo.ID)
	if err != nil {
		gCtx.HTML(http.StatusInternalServerError, "landing.html", gin.H{
			"error":   "Failed to create or find user",
			"baseURL": gCtx.Request.Host,
		})
		return
	}

	// Generate JWT token
	signedToken, err := GenerateJWT(user.ID, user.Email)
	if err != nil {
		gCtx.HTML(http.StatusInternalServerError, "landing.html", gin.H{
			"error":   "Failed to generate token",
			"baseURL": gCtx.Request.Host,
		})
		return
	}

	// Set the JWT as a secure cookie
	maxAge := int(config.AppConfig.JWTExpirationHours.Seconds())
	gCtx.SetCookie("jwt", signedToken, maxAge, "/", "", false, true)

	// Redirect to dashboard instead of home page
	gCtx.Redirect(http.StatusFound, "/dashboard")
}
