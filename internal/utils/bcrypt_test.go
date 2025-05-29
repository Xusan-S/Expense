package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashPassword(t *testing.T) {
	password := "password123"
	hashedPassword, err := HashPassword(password)

	assert.NoError(t, err)
	assert.NotEmpty(t, hashedPassword)
	assert.NotEqual(t, password, hashedPassword)
}

func TestCheckPasswordHash(t *testing.T) {
	password := "password123"
	hashedPassword, _ := HashPassword(password)

	assert.True(t, CheckPasswordHash(password, hashedPassword))
	assert.False(t, CheckPasswordHash("wrongpassword", hashedPassword))
}

func TestCheckPasswordHash_InvalidHash(t *testing.T) {
	assert.False(t, CheckPasswordHash("password123", "invalidhash"))
} 