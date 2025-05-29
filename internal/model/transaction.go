package model

import "time"

const (
	TransactionTypeIncome  = "income"
	TransactionTypeExpense = "expense"
)

// Transaction represents an income or expense record
type Transaction struct {
	ID              int64     `json:"id"`
	UserID          int       `json:"user_id"`
	Amount          int64     `json:"amount"` // In tiyns
	Type            string    `json:"type"`   // "income" or "expense"
	Category        string    `json:"category"`
	Description     *string   `json:"description,omitempty"` // Pointer for optional field
	TransactionDate time.Time `json:"transaction_date"`
	ReceiptPath     *string   `json:"receipt_path,omitempty"` // Pointer for optional field
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateTransactionRequest is used for creating a new transaction
type CreateTransactionRequest struct {
	Amount          int64     `json:"amount" binding:"required,gt=0"`
	Type            string    `json:"type" binding:"required,oneof=income expense"`
	Category        string    `json:"category" binding:"required"`
	Description     *string   `json:"description"`
	TransactionDate time.Time `json:"transaction_date"`
}

type UpdateTransactionRequest struct {
	Amount          *int64     `json:"amount,omitempty"` // Pointers to allow partial updates
	Type            *string    `json:"type,omitempty" binding:"omitempty,oneof=income expense"`
	Category        *string    `json:"category,omitempty"`
	Description     *string    `json:"description,omitempty"`
	TransactionDate *time.Time `json:"transaction_date,omitempty"`
}

// AdminTransactionFilter contains filter parameters for admin transaction queries
type AdminTransactionFilters struct {
	UserID    *int
	StartDate *time.Time
	EndDate   *time.Time
	Category  *string
	Type      *string
}

// UserTransactionFilter contains filter parameters for user transaction queries
type UserTransactionFilters struct {
	Type      *string
	Category  *string
	StartDate *time.Time // For filtering by date (start of day)
	EndDate   *time.Time // For filtering by date (end of day)
}

// AggregatedStats represents the statistics for admin
type AggregatedStats struct {
	TotalIncome       int64            `json:"total_income"`
	TotalExpenses     int64            `json:"total_expenses"`
	Balance           int64            `json:"balance"`
	ByCategoryIncome  map[string]int64 `json:"by_category_income"`
	ByCategoryExpense map[string]int64 `json:"by_category_expense"`
	ByUserSpending    map[int]UserStat `json:"by_user_spending"` // UserID -> Stats
}

type UserStat struct {
	UserID           int    `json:"user_id"`
	UserPhone        string `json:"user_phone"` // Added for easier display
	TotalSpent       int64  `json:"total_spent"`
	TotalIncome      int64  `json:"total_income"`
	TransactionCount int64  `json:"transaction_count"`
}
