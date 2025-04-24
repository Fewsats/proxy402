package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"linkshrink/internal/auth"
)

const (
	AuthorizationHeaderKey  = "Authorization"
	AuthorizationTypeBearer = "Bearer"
	AuthorizationPayloadKey = "authorization_payload"
)

// AuthMiddleware creates a Gin middleware for JWT authentication.
func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authorizationHeader := ctx.GetHeader(AuthorizationHeaderKey)
		if len(authorizationHeader) == 0 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is not provided"})
			return
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			return
		}

		authorizationType := fields[0]
		if authorizationType != AuthorizationTypeBearer {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unsupported authorization type"})
			return
		}

		accessToken := fields[1]
		payload, err := auth.ValidateJWT(accessToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Set the payload in the context for downstream handlers
		ctx.Set(AuthorizationPayloadKey, payload)
		ctx.Next()
	}
}
