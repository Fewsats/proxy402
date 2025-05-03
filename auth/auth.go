package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Service handles JWT generation and validation
type Service struct {
	config *Config
}

// NewService creates a new authentication service
func NewAuthService(config *Config) *Service {
	return &Service{
		config: config,
	}
}

// Claims defines the structure of the JWT claims.
type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// Service method implementations
// GenerateJWT creates a new JWT for a given user ID and email.
func (s *Service) GenerateJWT(userID uint, email string) (string, error) {
	expirationTime := time.Now().Add(s.config.JWTExpirationHours)
	claims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "linkshrink",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateJWT parses and validates a JWT string.
// It returns the claims if the token is valid, otherwise returns an error.
func (s *Service) ValidateJWT(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Ensure the signing method is HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, errors.New("token has expired")
		} else if errors.Is(err, jwt.ErrTokenMalformed) {
			return nil, errors.New("malformed token")
		} else if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, errors.New("invalid token signature")
		} else {
			return nil, fmt.Errorf("could not parse token: %w", err)
		}
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
