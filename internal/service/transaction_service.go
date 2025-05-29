package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"expense_tracker/internal/model"
	"expense_tracker/internal/repository"
)

var (
	ErrTransactionNotFound = errors.New("transaction not found")
	ErrForbidden           = errors.New("forbidden: user does not have permission for this action")
	ErrInvalidFileFormat   = errors.New("invalid file format. only .jpg, .png, .pdf are allowed")
	ErrFileSizeExceeded    = errors.New("file size exceeds limit")
)

const MaxFileSize = 5 * 1024 * 1024 // 5MB

// TransactionService defines operations for transactions
type TransactionService interface {
	CreateTransaction(ctx context.Context, userID int, req model.CreateTransactionRequest) (*model.Transaction, error)
	GetTransactionByID(ctx context.Context, transactionID int64, userID int, userRole string) (*model.Transaction, error)
	GetUserTransactions(ctx context.Context, userID int, filters model.UserTransactionFilters) ([]model.Transaction, error)
	UpdateTransaction(ctx context.Context, transactionID int64, userID int, req model.UpdateTransactionRequest) (*model.Transaction, error)
	DeleteTransaction(ctx context.Context, transactionID int64, userID int, userRole string) error
	UploadReceipt(ctx context.Context, transactionID int64, userID int, file *multipart.FileHeader, uploadsDir string) (*model.Transaction, error)
	GetReceiptPath(ctx context.Context, transactionID int64, userID int, userRole string) (string, string, error) // returns path and filename

	// Admin methods
	GetAllTransactionsAdmin(ctx context.Context, filters model.AdminTransactionFilters) ([]model.Transaction, error)
	GetStatisticsAdmin(ctx context.Context, filters model.AdminTransactionFilters) (*model.AggregatedStats, error)
	ExportTransactionsCSVAdmin(ctx context.Context, filters model.AdminTransactionFilters) (*bytes.Buffer, error)
}

type transactionService struct {
	repo       repository.TransactionRepository
	uploadsDir string
}

// NewTransactionService creates a new TransactionService
func NewTransactionService(repo repository.TransactionRepository, uploadsDir string) TransactionService {
	return &transactionService{repo: repo, uploadsDir: uploadsDir}
}

func (s *transactionService) CreateTransaction(ctx context.Context, userID int, req model.CreateTransactionRequest) (*model.Transaction, error) {
	transactionDate := req.TransactionDate
	if transactionDate.IsZero() {
		transactionDate = time.Now()
	}

	transaction := &model.Transaction{
		UserID:          userID,
		Amount:          req.Amount,
		Type:            req.Type,
		Category:        req.Category,
		Description:     req.Description,
		TransactionDate: transactionDate,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.repo.Create(ctx, transaction); err != nil {
		return nil, fmt.Errorf("failed to create transaction in repo: %w", err)
	}
	return transaction, nil
}

func (s *transactionService) GetTransactionByID(ctx context.Context, transactionID int64, userID int, userRole string) (*model.Transaction, error) {
	transaction, err := s.repo.FindByID(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find transaction by ID: %w", err)
	}
	if transaction == nil {
		return nil, ErrTransactionNotFound
	}

	if userRole != model.RoleAdmin && transaction.UserID != userID {
		return nil, ErrForbidden
	}
	return transaction, nil
}

func (s *transactionService) GetUserTransactions(ctx context.Context, userID int, filters model.UserTransactionFilters) ([]model.Transaction, error) {
	// Adjust date filters to cover the whole day if only date part is provided
	if filters.StartDate != nil && filters.StartDate.Hour() == 0 && filters.StartDate.Minute() == 0 && filters.StartDate.Second() == 0 {
		startOfDay := time.Date(filters.StartDate.Year(), filters.StartDate.Month(), filters.StartDate.Day(), 0, 0, 0, 0, filters.StartDate.Location())
		filters.StartDate = &startOfDay
		if filters.EndDate == nil { // If only start date (a specific day) is given, set end date to end of that day
			endOfDay := time.Date(filters.StartDate.Year(), filters.StartDate.Month(), filters.StartDate.Day(), 23, 59, 59, 999999999, filters.StartDate.Location())
			filters.EndDate = &endOfDay
		}
	}
	if filters.EndDate != nil && filters.EndDate.Hour() == 0 && filters.EndDate.Minute() == 0 && filters.EndDate.Second() == 0 {
		endOfDay := time.Date(filters.EndDate.Year(), filters.EndDate.Month(), filters.EndDate.Day(), 23, 59, 59, 999999999, filters.EndDate.Location())
		filters.EndDate = &endOfDay
	}

	transactions, err := s.repo.FindByUser(ctx, userID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get user transactions from repo: %w", err)
	}
	return transactions, nil
}

func (s *transactionService) UpdateTransaction(ctx context.Context, transactionID int64, userID int, req model.UpdateTransactionRequest) (*model.Transaction, error) {
	existingTx, err := s.repo.FindByID(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find transaction for update: %w", err)
	}
	if existingTx == nil {
		return nil, ErrTransactionNotFound
	}
	if existingTx.UserID != userID { // Only author can edit
		return nil, ErrForbidden
	}

	// Apply updates
	if req.Amount != nil {
		existingTx.Amount = *req.Amount
	}
	if req.Type != nil {
		existingTx.Type = *req.Type
	}
	if req.Category != nil {
		existingTx.Category = *req.Category
	}
	if req.Description != nil { // handles setting to "" or null
		existingTx.Description = req.Description
	}
	if req.TransactionDate != nil {
		existingTx.TransactionDate = *req.TransactionDate
	}
	existingTx.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, existingTx); err != nil {
		return nil, fmt.Errorf("failed to update transaction in repo: %w", err)
	}
	return existingTx, nil
}

func (s *transactionService) DeleteTransaction(ctx context.Context, transactionID int64, userID int, userRole string) error {
	existingTx, err := s.repo.FindByID(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("failed to find transaction for deletion: %w", err)
	}
	if existingTx == nil {
		return ErrTransactionNotFound
	}

	if userRole != model.RoleAdmin && existingTx.UserID != userID {
		return ErrForbidden
	}
	if err := s.repo.Delete(ctx, transactionID); err != nil {
		return fmt.Errorf("failed to delete transaction in repo: %w", err)
	}
	return nil
}

func (s *transactionService) UploadReceipt(ctx context.Context, transactionID int64, userID int, fileHeader *multipart.FileHeader, baseUploadsDir string) (*model.Transaction, error) {
	transaction, err := s.repo.FindByID(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find transaction for receipt upload: %w", err)
	}
	if transaction == nil {
		return nil, ErrTransactionNotFound
	}
	if transaction.UserID != userID { // Only author can upload receipt for their transaction
		return nil, ErrForbidden
	}

	// Validate file
	if fileHeader.Size > MaxFileSize {
		return nil, ErrFileSizeExceeded
	}
	ext := filepath.Ext(fileHeader.Filename)
	allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".pdf": true}
	if !allowedExts[strings.ToLower(ext)] {
		return nil, ErrInvalidFileFormat
	}

	transactionUploadsDir := filepath.Join(baseUploadsDir, "transactions", strconv.FormatInt(transactionID, 10))
	if err := os.MkdirAll(transactionUploadsDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	fileName := filepath.Base(fileHeader.Filename) // Basic sanitization
	filePath := filepath.Join(transactionUploadsDir, fileName)
	relativeFilePath := filepath.ToSlash(filePath) // Store with forward slashes for consistency

	// Save the file
	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file on server: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Update transaction with receipt path
	if err := s.repo.UpdateReceiptPath(ctx, transactionID, relativeFilePath); err != nil {
		// Potentially roll back file save or log, then return error
		os.Remove(filePath) // Attempt to clean up
		return nil, fmt.Errorf("failed to update transaction with receipt path: %w", err)
	}

	transaction.ReceiptPath = &relativeFilePath // Update the model in memory
	return transaction, nil
}

func (s *transactionService) GetReceiptPath(ctx context.Context, transactionID int64, userID int, userRole string) (string, string, error) {
	transaction, err := s.repo.FindByID(ctx, transactionID)
	if err != nil {
		return "", "", fmt.Errorf("failed to find transaction for receipt retrieval: %w", err)
	}
	if transaction == nil {
		return "", "", ErrTransactionNotFound
	}

	if userRole != model.RoleAdmin && transaction.UserID != userID {
		return "", "", ErrForbidden
	}

	if transaction.ReceiptPath == nil || *transaction.ReceiptPath == "" {
		return "", "", fmt.Errorf("receipt not found for this transaction")
	}

	fullPath := filepath.FromSlash(*transaction.ReceiptPath)
	fileName := filepath.Base(fullPath)

	return fullPath, fileName, nil
}

// --- Admin Methods ---

func (s *transactionService) GetAllTransactionsAdmin(ctx context.Context, filters model.AdminTransactionFilters) ([]model.Transaction, error) {
	transactions, err := s.repo.FindAll(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get all transactions for admin: %w", err)
	}
	return transactions, nil
}

func (s *transactionService) GetStatisticsAdmin(ctx context.Context, filters model.AdminTransactionFilters) (*model.AggregatedStats, error) {
	stats, err := s.repo.GetAggregatedStats(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregated stats for admin: %w", err)
	}
	return stats, nil
}

func (s *transactionService) ExportTransactionsCSVAdmin(ctx context.Context, filters model.AdminTransactionFilters) (*bytes.Buffer, error) {
	transactions, err := s.repo.FindAll(ctx, filters) // Use FindAll which already supports AdminTransactionFilters
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions for CSV export: %w", err)
	}

	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)

	// Write header
	header := []string{"ID", "UserID", "Amount", "Type", "Category", "Description", "TransactionDate", "CreatedAt", "ReceiptPath"}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write rows
	for _, t := range transactions {
		var desc, receiptPath string
		if t.Description != nil {
			desc = *t.Description
		}
		if t.ReceiptPath != nil {
			receiptPath = *t.ReceiptPath
		}
		row := []string{
			strconv.FormatInt(t.ID, 10),
			strconv.Itoa(t.UserID),
			strconv.FormatInt(t.Amount, 10), // Amount in cents
			t.Type,
			t.Category,
			desc,
			t.TransactionDate.Format(time.RFC3339),
			t.CreatedAt.Format(time.RFC3339),
			receiptPath,
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("error flushing CSV writer: %w", err)
	}

	return buffer, nil
}
