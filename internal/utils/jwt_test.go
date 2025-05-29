package utils

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestJWTUtil_GenerateToken(t *testing.T) {
	jwtUtil := NewJWTUtil("secret", 1)
	userID := 1
	role := "user"

	tokenString, err := jwtUtil.GenerateToken(userID, role)

	assert.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	// Validate the token to ensure it's well-formed and contains correct claims
	claims, err := jwtUtil.ValidateToken(tokenString)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, role, claims.Role)
	assert.WithinDuration(t, time.Now().Add(time.Hour), claims.ExpiresAt.Time, 5*time.Second)
}

func TestJWTUtil_ValidateToken(t *testing.T) {
	jwtUtil := NewJWTUtil("secret", 1)
	userID := 1
	role := "user"

	tokenString, _ := jwtUtil.GenerateToken(userID, role)

	claims, err := jwtUtil.ValidateToken(tokenString)

	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, role, claims.Role)
}

func TestJWTUtil_ValidateToken_InvalidToken(t *testing.T) {
	jwtUtil := NewJWTUtil("secret", 1)

	_, err := jwtUtil.ValidateToken("invalid.token.string")
	assert.Error(t, err)
}

func TestJWTUtil_ValidateToken_ExpiredToken(t *testing.T) {
	jwtUtil := NewJWTUtil("secret", -1) // Token expires in the past
	userID := 1
	role := "user"

	tokenString, _ := jwtUtil.GenerateToken(userID, role)

	// Wait for a moment to ensure the token is definitely expired if system clock is slightly off
	time.Sleep(1 * time.Second)

	_, err := jwtUtil.ValidateToken(tokenString)
	assert.Error(t, err)
	// Check for specific error related to token expiration if library provides it
	// For github.com/golang-jwt/jwt/v5, a common error is jwt.ErrTokenExpired
	assert.ErrorIs(t, err, jwt.ErrTokenExpired)
}

func TestJWTUtil_ValidateToken_WrongSecret(t *testing.T) {
	jwtUtil1 := NewJWTUtil("secret1", 1)
	jwtUtil2 := NewJWTUtil("secret2", 1)
	userID := 1
	role := "user"

	tokenString, _ := jwtUtil1.GenerateToken(userID, role)

	_, err := jwtUtil2.ValidateToken(tokenString)
	assert.Error(t, err)
}

func TestJWTUtil_ValidateToken_InvalidSigningMethod(t *testing.T) {
	jwtUtil := NewJWTUtil("secret", 1)
	// Create a token with a different signing method (e.g., HS384 instead of HS256)
	claims := &JWTClaims{
		UserID: 1,
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
	// Sign with the same secret, as the key type is compatible for HMAC algorithms
	tokenString, _ := token.SignedString([]byte("secret"))

	_, err := jwtUtil.ValidateToken(tokenString)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected signing method")
} 