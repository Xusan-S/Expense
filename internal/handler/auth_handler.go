package handler

import (
	"errors"
	"net/http"

	//"expense_tracker/internal/model"
	"expense_tracker/internal/service"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	service service.AuthService
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(s service.AuthService) *AuthHandler {
	return &AuthHandler{service: s}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Phone    string `json:"phone" binding:"required"`
		Password string `json:"password" binding:"required,min=6"` // Basic validation
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	user, token, err := h.service.Register(c.Request.Context(), req.Phone, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		// Log the detailed error for server admins
		// log.Printf("Error during registration: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user_id": user.ID,
		"phone":   user.Phone,
		"role":    user.Role,
		"token":   token,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Phone    string `json:"phone" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	user, token, err := h.service.Login(c.Request.Context(), req.Phone, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) || errors.Is(err, service.ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": service.ErrInvalidCredentials.Error()})
			return
		}
		// log.Printf("Error during login: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to login"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"user_id": user.ID,
		"phone":   user.Phone,
		"role":    user.Role,
		"token":   token,
	})
}

// RegisterAuthRoutes registers auth routes
func (h *AuthHandler) RegisterAuthRoutes(rg *gin.RouterGroup) {
	authGroup := rg.Group("/auth")
	{
		authGroup.POST("/register", h.Register)
		authGroup.POST("/login", h.Login)
	}
}
