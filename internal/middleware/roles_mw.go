package middleware

import (
	"net/http"

	"expense_tracker/internal/model"

	"github.com/gin-gonic/gin"
)

// RoleMiddleware creates a middleware to check for specific user roles
func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get(AuthRoleKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Role not found in token, ensure JWT middleware runs first"})
			return
		}

		userRole, ok := roleVal.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Invalid role type in token"})
			return
		}

		isAllowed := false
		for _, allowedRole := range allowedRoles {
			if userRole == allowedRole {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "You do not have permission to access this resource"})
			return
		}

		c.Next()
	}
}

// AdminMiddleware checks if the user is an admin
func AdminMiddleware() gin.HandlerFunc {
	return RoleMiddleware(model.RoleAdmin)
}

// UserMiddleware checks if the user has the 'user' role \
func UserMiddleware() gin.HandlerFunc {
	return RoleMiddleware(model.RoleUser, model.RoleAdmin) // Allows both users and admins for "user" routes
}

// StrictlyUserMiddleware checks if the user has only 'user' role
func StrictlyUserMiddleware() gin.HandlerFunc {
	return RoleMiddleware(model.RoleUser)
}
