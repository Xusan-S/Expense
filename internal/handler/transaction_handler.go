package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	///"path/filepath"
	"strconv"
	"strings"
	"time"

	"expense_tracker/internal/middleware"
	"expense_tracker/internal/model"
	"expense_tracker/internal/service"

	"github.com/gin-gonic/gin"
)

// TransactionHandler handles transaction related requests
type TransactionHandler struct {
	service    service.TransactionService
	uploadsDir string
}

// NewTransactionHandler creates a new TransactionHandler
func NewTransactionHandler(s service.TransactionService, uploadsDir string) *TransactionHandler {
	return &TransactionHandler{service: s, uploadsDir: uploadsDir}
}

// Helper to get authenticated user ID from context
func getAuthUserID(c *gin.Context) (int, error) {
	userIDVal, exists := c.Get(middleware.AuthUserKey)
	if !exists {
		return 0, errors.New("user ID not found in context")
	}
	userID, ok := userIDVal.(int)
	if !ok {
		return 0, errors.New("invalid user ID type in context")
	}
	return userID, nil
}

// Helper to get authenticated user role from context
func getAuthUserRole(c *gin.Context) (string, error) {
	roleVal, exists := c.Get(middleware.AuthRoleKey)
	if !exists {
		return "", errors.New("user role not found in context")
	}
	role, ok := roleVal.(string)
	if !ok {
		return "", errors.New("invalid user role type in context")
	}
	return role, nil
}

func (h *TransactionHandler) CreateTransaction(c *gin.Context) {
	userID, err := getAuthUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req model.CreateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	transaction, err := h.service.CreateTransaction(c.Request.Context(), userID, req)
	if err != nil {
		log.Printf("Error creating transaction: %v", err) // Log detailed error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction"})
		return
	}
	c.JSON(http.StatusCreated, transaction)
}

func (h *TransactionHandler) GetMyTransactions(c *gin.Context) {
	userID, err := getAuthUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var filters model.UserTransactionFilters
	if typeParam := c.Query("type"); typeParam != "" {
		filters.Type = &typeParam
	}
	if categoryParam := c.Query("category"); categoryParam != "" {
		filters.Category = &categoryParam
	}
	if dateParam := c.Query("date"); dateParam != "" {
		parsedDate, err := time.Parse("2006-01-02", dateParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format for 'date', use YYYY-MM-DD"})
			return
		}
		// Set StartDate to the beginning of the parsed day and EndDate to the end of the parsed day
		loc := time.Local // Or UTC, depending on desired behavior
		startOfDay := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 0, 0, 0, 0, loc)
		endOfDay := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 23, 59, 59, 999999999, loc)
		filters.StartDate = &startOfDay
		filters.EndDate = &endOfDay
	}

	transactions, err := h.service.GetUserTransactions(c.Request.Context(), userID, filters)
	if err != nil {
		log.Printf("Error getting user transactions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve transactions"})
		return
	}
	c.JSON(http.StatusOK, transactions)
}

func (h *TransactionHandler) GetTransactionByID(c *gin.Context) {
	userID, err := getAuthUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	userRole, err := getAuthUserRole(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
		return
	}

	transactionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	transaction, err := h.service.GetTransactionByID(c.Request.Context(), transactionID, userID, userRole)
	if err != nil {
		if errors.Is(err, service.ErrTransactionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else if errors.Is(err, service.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else {
			log.Printf("Error getting transaction by ID: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve transaction"})
		}
		return
	}
	c.JSON(http.StatusOK, transaction)
}

func (h *TransactionHandler) UpdateTransaction(c *gin.Context) {
	userID, err := getAuthUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	transactionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	var req model.UpdateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	transaction, err := h.service.UpdateTransaction(c.Request.Context(), transactionID, userID, req)
	if err != nil {
		if errors.Is(err, service.ErrTransactionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else if errors.Is(err, service.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else {
			log.Printf("Error updating transaction: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction"})
		}
		return
	}
	c.JSON(http.StatusOK, transaction)
}

func (h *TransactionHandler) DeleteTransaction(c *gin.Context) {
	userID, err := getAuthUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	userRole, err := getAuthUserRole(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
		return
	}

	transactionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	err = h.service.DeleteTransaction(c.Request.Context(), transactionID, userID, userRole)
	if err != nil {
		if errors.Is(err, service.ErrTransactionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else if errors.Is(err, service.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else {
			log.Printf("Error deleting transaction: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete transaction"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Transaction deleted successfully"})
}

// --- Receipt Handling ---

func (h *TransactionHandler) UploadReceipt(c *gin.Context) {
	userID, err := getAuthUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required: " + err.Error()})
		return
	}

	transactionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	file, err := c.FormFile("receipt")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Receipt file is required: " + err.Error()})
		return
	}

	updatedTransaction, err := h.service.UploadReceipt(c.Request.Context(), transactionID, userID, file, h.uploadsDir)
	if err != nil {
		if errors.Is(err, service.ErrTransactionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else if errors.Is(err, service.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else if errors.Is(err, service.ErrInvalidFileFormat) || errors.Is(err, service.ErrFileSizeExceeded) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			log.Printf("Error uploading receipt: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload receipt"})
		}
		return
	}
	c.JSON(http.StatusOK, updatedTransaction)
}

func (h *TransactionHandler) GetReceipt(c *gin.Context) {
	userID, err := getAuthUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required: " + err.Error()})
		return
	}
	userRole, err := getAuthUserRole(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
		return
	}

	transactionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	filePath, fileName, err := h.service.GetReceiptPath(c.Request.Context(), transactionID, userID, userRole)
	if err != nil {
		if errors.Is(err, service.ErrTransactionNotFound) || strings.Contains(err.Error(), "receipt not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else if errors.Is(err, service.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else {
			log.Printf("Error getting receipt path: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get receipt path"})
		}
		return
	}

	// Check if file exists before attempting to serve
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Receipt file not found on server"})
		return
	}

	c.FileAttachment(filePath, fileName)
}

// --- Admin Routes ---

func (h *TransactionHandler) GetAllTransactionsAdmin(c *gin.Context) {
	var filters model.AdminTransactionFilters
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		uid, err := strconv.Atoi(userIDStr)
		if err == nil {
			filters.UserID = &uid
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id format"})
			return
		}
	}
	if typeParam := c.Query("type"); typeParam != "" {
		filters.Type = &typeParam
	}
	if categoryParam := c.Query("category"); categoryParam != "" {
		filters.Category = &categoryParam
	}
	if startDateParam := c.Query("start_date"); startDateParam != "" {
		parsedDate, err := time.Parse("2006-01-02", startDateParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format for 'start_date', use YYYY-MM-DD"})
			return
		}
		filters.StartDate = &parsedDate
	}
	if endDateParam := c.Query("end_date"); endDateParam != "" {
		parsedDate, err := time.Parse("2006-01-02", endDateParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format for 'end_date', use YYYY-MM-DD"})
			return
		}
		// Adjust end date to include the whole day
		endOfDay := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 23, 59, 59, 999999999, parsedDate.Location())
		filters.EndDate = &endOfDay
	}

	transactions, err := h.service.GetAllTransactionsAdmin(c.Request.Context(), filters)
	if err != nil {
		log.Printf("Error getting all transactions for admin: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve transactions"})
		return
	}
	c.JSON(http.StatusOK, transactions)
}

func (h *TransactionHandler) GetStatisticsAdmin(c *gin.Context) {
	var filters model.AdminTransactionFilters // Same filters as GetAllTransactionsAdmin
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		uid, err := strconv.Atoi(userIDStr)
		if err == nil {
			filters.UserID = &uid
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id format"})
			return
		}
	}
	if typeParam := c.Query("type"); typeParam != "" {
		filters.Type = &typeParam
	}
	if categoryParam := c.Query("category"); categoryParam != "" {
		filters.Category = &categoryParam
	}
	if startDateParam := c.Query("start_date"); startDateParam != "" {
		parsedDate, err := time.Parse("2006-01-02", startDateParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format for 'start_date', use YYYY-MM-DD"})
			return
		}
		filters.StartDate = &parsedDate
	}
	if endDateParam := c.Query("end_date"); endDateParam != "" {
		parsedDate, err := time.Parse("2006-01-02", endDateParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format for 'end_date', use YYYY-MM-DD"})
			return
		}
		endOfDay := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 23, 59, 59, 999999999, parsedDate.Location())
		filters.EndDate = &endOfDay
	}

	stats, err := h.service.GetStatisticsAdmin(c.Request.Context(), filters)
	if err != nil {
		log.Printf("Error getting statistics for admin: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve statistics"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *TransactionHandler) ExportTransactionsCSVAdmin(c *gin.Context) {
	var filters model.AdminTransactionFilters // Same filters as GetAllTransactionsAdmin
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		uid, err := strconv.Atoi(userIDStr)
		if err == nil {
			filters.UserID = &uid
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user_id format"})
			return
		}
	}
	if typeParam := c.Query("type"); typeParam != "" {
		filters.Type = &typeParam
	}
	if categoryParam := c.Query("category"); categoryParam != "" {
		filters.Category = &categoryParam
	}
	if startDateParam := c.Query("start_date"); startDateParam != "" {
		parsedDate, err := time.Parse("2006-01-02", startDateParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format for 'start_date', use YYYY-MM-DD"})
			return
		}
		filters.StartDate = &parsedDate
	}
	if endDateParam := c.Query("end_date"); endDateParam != "" {
		parsedDate, err := time.Parse("2006-01-02", endDateParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format for 'end_date', use YYYY-MM-DD"})
			return
		}
		endOfDay := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 23, 59, 59, 999999999, parsedDate.Location())
		filters.EndDate = &endOfDay
	}

	csvBuffer, err := h.service.ExportTransactionsCSVAdmin(c.Request.Context(), filters)
	if err != nil {
		log.Printf("Error exporting transactions to CSV for admin: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export transactions to CSV"})
		return
	}

	fileName := fmt.Sprintf("transactions_export_%s.csv", time.Now().Format("20060102_150405"))
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Data(http.StatusOK, "text/csv", csvBuffer.Bytes())
}

// RegisterTransactionRoutes registers transaction routes
func (h *TransactionHandler) RegisterTransactionRoutes(rg *gin.RouterGroup, authMW gin.HandlerFunc, userMW gin.HandlerFunc, adminMW gin.HandlerFunc) {
	// User-specific transaction routes (requires auth, any authenticated user)
	userTxRoutes := rg.Group("/transactions")
	userTxRoutes.Use(authMW) // All routes in this group require authentication
	// userTxRoutes.Use(userMW) // Further ensures it's a user or admin acting as user
	{
		userTxRoutes.POST("", h.CreateTransaction)
		userTxRoutes.GET("", h.GetMyTransactions)
		userTxRoutes.GET("/:id", h.GetTransactionByID)     // Service layer handles ownership for non-admins
		userTxRoutes.PUT("/:id", h.UpdateTransaction)      // Service layer handles ownership
		userTxRoutes.DELETE("/:id", h.DeleteTransaction)   // Service layer handles ownership for non-admins
		userTxRoutes.POST("/:id/receipt", h.UploadReceipt) // Service layer handles ownership
		userTxRoutes.GET("/:id/receipt", h.GetReceipt)     // Service layer handles ownership for non-admins
	}

	// Admin-specific transaction routes
	adminRoutes := rg.Group("/admin")
	adminRoutes.Use(authMW)  // Requires authentication
	adminRoutes.Use(adminMW) // Requires admin role
	{
		adminRoutes.GET("/transactions", h.GetAllTransactionsAdmin)
		adminRoutes.GET("/stats", h.GetStatisticsAdmin)
		adminRoutes.GET("/transactions/export/csv", h.ExportTransactionsCSVAdmin)
	}
}
