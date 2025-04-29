package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"linkshrink/internal/auth"
)

const (
	AuthorizationPayloadKey = "authorization_payload"
	UserKey                 = "user"
	JWTCookie               = "jwt"
)

// AuthMiddleware creates a Gin middleware for JWT authentication.
func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Get JWT from cookie
		token, err := ctx.Cookie(JWTCookie)

		// For root path, render without user data if no valid token
		if err != nil || token == "" {
			if ctx.FullPath() == "/" {
				ctx.Next()
				return
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}

		// Validate the token
		claims, err := auth.ValidateJWT(token)
		if err != nil {
			// Clear invalid cookie
			ctx.SetCookie(JWTCookie, "", -1, "/", "", false, true)

			if ctx.FullPath() == "/" {
				ctx.Next()
				return
			}

			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
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
