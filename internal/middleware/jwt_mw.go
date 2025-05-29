package middleware

import (
	"net/http"
	"strings"

	"expense_tracker/internal/utils"

	"github.com/gin-gonic/gin"
)

const (
	AuthUserKey = "authUser"
	AuthRoleKey = "authRole"
)

// JWTAuthMiddleware creates a middleware for JWT authentication
func JWTAuthMiddleware(jwtUtil *utils.JWTUtil) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			return
		}

		tokenString := parts[1]
		claims, err := jwtUtil.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// Set user information in context
		c.Set(AuthUserKey, claims.UserID)
		c.Set(AuthRoleKey, claims.Role)

		c.Next()
	}
}
