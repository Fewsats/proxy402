package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	AuthorizationPayloadKey = "authorization_payload"
	UserKey                 = "user"
	JWTCookie               = "jwt"
)

// Authenticator is the interface for auth services that can authenticate requests
type Authenticator interface {
	ValidateJWT(token string) (*Claims, error)
}

// AuthMiddleware creates a Gin middleware for JWT authentication.
func AuthMiddleware(auth Authenticator) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Get JWT from cookie
		token, err := ctx.Cookie(JWTCookie)

		// Redirect to landing page if no valid token
		if err != nil || token == "" {
			// Special handling for root path was here but no longer needed
			// since we have separate routes now
			ctx.Redirect(http.StatusFound, "/")
			ctx.Abort()
			return
		}

		// Validate the token
		claims, err := auth.ValidateJWT(token)
		if err != nil {
			// Clear invalid cookie
			ctx.SetCookie(JWTCookie, "", -1, "/", "", false, true)

			// Redirect to landing on invalid token
			ctx.Redirect(http.StatusFound, "/")
			ctx.Abort()
			return
		}

		// Set user info in the context for downstream handlers
		ctx.Set(AuthorizationPayloadKey, claims)
		ctx.Set(UserKey, gin.H{
			"id":    claims.UserID,
			"email": claims.Email,
		})
		ctx.Next()
	}
}

// GetUser returns the user information from the gin context
func GetUser(ctx *gin.Context) (gin.H, bool) {
	user, exists := ctx.Get(UserKey)
	if !exists {
		return nil, false
	}

	return user.(gin.H), true
}

// GetUserID returns the user ID from the gin context
func GetUserID(ctx *gin.Context) (uint, bool) {
	user, exists := GetUser(ctx)
	if !exists {
		return 0, false
	}

	return uint(user["id"].(uint)), true
}
