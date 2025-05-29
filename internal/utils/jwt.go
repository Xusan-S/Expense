package utils

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims custom claims for JWT
type JWTClaims struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWTUtil provides JWT generation and validation
type JWTUtil struct {
	secretKey      string
	expirationHours int64
}

// NewJWTUtil creates a new JWTUtil
func NewJWTUtil(secretKey string, expirationHours int64) *JWTUtil {
	return &JWTUtil{secretKey: secretKey, expirationHours: expirationHours}
}

// GenerateToken generates a new JWT token
func (ju *JWTUtil) GenerateToken(userID int, role string) (string, error) {
	claims := &JWTClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(ju.expirationHours))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   strconv.Itoa(userID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(ju.secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return tokenString, nil
}

// ValidateToken validates the JWT token
func (ju *JWTUtil) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(ju.secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}