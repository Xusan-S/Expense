package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"expense_tracker/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TransactionRepository defines operations for transaction data
type TransactionRepository interface {
	Create(ctx context.Context, transaction *model.Transaction) error
	FindByID(ctx context.Context, id int64) (*model.Transaction, error)
	FindByUser(ctx context.Context, userID int, filters model.UserTransactionFilters) ([]model.Transaction, error)
	Update(ctx context.Context, transaction *model.Transaction) error
	Delete(ctx context.Context, id int64) error
	UpdateReceiptPath(ctx context.Context, id int64, receiptPath string) error
	FindAll(ctx context.Context, filters model.AdminTransactionFilters) ([]model.Transaction, error)
	GetAggregatedStats(ctx context.Context, filters model.AdminTransactionFilters) (*model.AggregatedStats, error)
}

type transactionRepository struct {
	db *pgxpool.Pool
}

// NewTransactionRepository creates a new TransactionRepository
func NewTransactionRepository(db *pgxpool.Pool) TransactionRepository {
	return &transactionRepository{db: db}
}

// Create inserts a new transaction into the database
func (r *transactionRepository) Create(ctx context.Context, t *model.Transaction) error {
	sql := `INSERT INTO transactions (user_id, amount, type, category, description, transaction_date, receipt_path, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id, created_at, updated_at`
	err := r.db.QueryRow(ctx, sql, t.UserID, t.Amount, t.Type, t.Category, t.Description, t.TransactionDate, t.ReceiptPath, t.CreatedAt, t.UpdatedAt).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	return nil
}

// FindByID retrieves a transaction by its ID
func (r *transactionRepository) FindByID(ctx context.Context, id int64) (*model.Transaction, error) {
	t := &model.Transaction{}
	sql := `SELECT id, user_id, amount, type, category, description, transaction_date, receipt_path, created_at, updated_at 
            FROM transactions WHERE id = $1`
	err := r.db.QueryRow(ctx, sql, id).Scan(
		&t.ID, &t.UserID, &t.Amount, &t.Type, &t.Category, &t.Description,
		&t.TransactionDate, &t.ReceiptPath, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to find transaction by ID: %w", err)
	}
	return t, nil
}

// FindByUser retrieves transactions for a specific user with optional filters
func (r *transactionRepository) FindByUser(ctx context.Context, userID int, filters model.UserTransactionFilters) ([]model.Transaction, error) {
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`SELECT id, user_id, amount, type, category, description, transaction_date, receipt_path, created_at, updated_at 
                               FROM transactions WHERE user_id = $1`)
	args := []interface{}{userID}
	argCount := 2 // Start after user_id

	if filters.Type != nil && *filters.Type != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND type = $%d", argCount))
		args = append(args, *filters.Type)
		argCount++
	}
	if filters.Category != nil && *filters.Category != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND category = $%d", argCount))
		args = append(args, *filters.Category)
		argCount++
	}
	if filters.StartDate != nil {
		queryBuilder.WriteString(fmt.Sprintf(" AND transaction_date >= $%d", argCount))
		args = append(args, *filters.StartDate)
		argCount++
	}
	if filters.EndDate != nil {
		queryBuilder.WriteString(fmt.Sprintf(" AND transaction_date <= $%d", argCount))
		args = append(args, *filters.EndDate)
		//argCount++
	}

	queryBuilder.WriteString(" ORDER BY transaction_date DESC, created_at DESC")

	rows, err := r.db.Query(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions by user: %w", err)
	}
	defer rows.Close()

	var transactions []model.Transaction
	for rows.Next() {
		var t model.Transaction
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.Amount, &t.Type, &t.Category, &t.Description,
			&t.TransactionDate, &t.ReceiptPath, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan transaction row: %w", err)
		}
		transactions = append(transactions, t)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction rows: %w", err)
	}
	return transactions, nil
}

// Update modifies an existing transaction
func (r *transactionRepository) Update(ctx context.Context, t *model.Transaction) error {
	sql := `UPDATE transactions 
            SET amount = $1, type = $2, category = $3, description = $4, transaction_date = $5, updated_at = NOW()
            WHERE id = $6 AND user_id = $7 RETURNING updated_at` // ensure user_id matches for ownership
	err := r.db.QueryRow(ctx, sql, t.Amount, t.Type, t.Category, t.Description, t.TransactionDate, t.ID, t.UserID).Scan(&t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("transaction not found or not owned by user for update")
		}
		return fmt.Errorf("failed to update transaction: %w", err)
	}
	return nil
}

// Delete removes a transaction from the database
func (r *transactionRepository) Delete(ctx context.Context, id int64) error {

	sql := `DELETE FROM transactions WHERE id = $1`
	cmdTag, err := r.db.Exec(ctx, sql, id)
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("transaction not found for deletion") // Or handle as non-error if idempotent delete is ok
	}
	return nil
}

// UpdateReceiptPath updates the receipt path for a transaction
func (r *transactionRepository) UpdateReceiptPath(ctx context.Context, id int64, receiptPath string) error {
	sql := `UPDATE transactions SET receipt_path = $1, updated_at = NOW() WHERE id = $2 RETURNING updated_at`
	var updatedAt time.Time
	err := r.db.QueryRow(ctx, sql, receiptPath, id).Scan(&updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("transaction not found for receipt path update")
		}
		return fmt.Errorf("failed to update receipt path: %w", err)
	}
	return nil
}

// FindAll retrieves all transactions with optional filters for admin
func (r *transactionRepository) FindAll(ctx context.Context, filters model.AdminTransactionFilters) ([]model.Transaction, error) {
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`SELECT t.id, t.user_id, t.amount, t.type, t.category, t.description, t.transaction_date, t.receipt_path, t.created_at, t.updated_at 
                               FROM transactions t`)

	args := []interface{}{}
	argCount := 1
	var conditions []string

	if filters.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("t.user_id = $%d", argCount))
		args = append(args, *filters.UserID)
		argCount++
	}
	if filters.Type != nil && *filters.Type != "" {
		conditions = append(conditions, fmt.Sprintf("t.type = $%d", argCount))
		args = append(args, *filters.Type)
		argCount++
	}
	if filters.Category != nil && *filters.Category != "" {
		conditions = append(conditions, fmt.Sprintf("t.category = $%d", argCount))
		args = append(args, *filters.Category)
		argCount++
	}
	if filters.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("t.transaction_date >= $%d", argCount))
		args = append(args, *filters.StartDate)
		argCount++
	}
	if filters.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("t.transaction_date <= $%d", argCount))
		args = append(args, *filters.EndDate)
		//argCount++
	}

	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(conditions, " AND "))
	}
	queryBuilder.WriteString(" ORDER BY t.transaction_date DESC, t.created_at DESC")

	rows, err := r.db.Query(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query all transactions: %w", err)
	}
	defer rows.Close()

	var transactions []model.Transaction
	for rows.Next() {
		var t model.Transaction
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.Amount, &t.Type, &t.Category, &t.Description,
			&t.TransactionDate, &t.ReceiptPath, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan transaction row for admin: %w", err)
		}
		transactions = append(transactions, t)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating admin transaction rows: %w", err)
	}
	return transactions, nil
}

// GetAggregatedStats calculates aggregated statistics for admin
func (r *transactionRepository) GetAggregatedStats(ctx context.Context, filters model.AdminTransactionFilters) (*model.AggregatedStats, error) {
	stats := &model.AggregatedStats{
		ByCategoryIncome:  make(map[string]int64),
		ByCategoryExpense: make(map[string]int64),
		ByUserSpending:    make(map[int]model.UserStat),
	}

	var baseQuery strings.Builder
	baseQuery.WriteString("FROM transactions t JOIN users u ON t.user_id = u.id")

	args := []interface{}{}
	argCount := 1
	var conditions []string

	if filters.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("t.user_id = $%d", argCount))
		args = append(args, *filters.UserID)
		argCount++
	}
	// Apply type filter if specified, otherwise it applies to income/expense specific queries
	// For overall stats, type filter might be restrictive but let's allow it.
	if filters.Type != nil && *filters.Type != "" {
		conditions = append(conditions, fmt.Sprintf("t.type = $%d", argCount))
		args = append(args, *filters.Type)
		argCount++
	}
	if filters.Category != nil && *filters.Category != "" {
		conditions = append(conditions, fmt.Sprintf("t.category = $%d", argCount))
		args = append(args, *filters.Category)
		argCount++
	}
	if filters.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("t.transaction_date >= $%d", argCount))
		args = append(args, *filters.StartDate)
		argCount++
	}
	if filters.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("t.transaction_date <= $%d", argCount))
		args = append(args, *filters.EndDate)
		argCount++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Total Income and Expenses
	sumQuery := fmt.Sprintf(`
        SELECT 
            COALESCE(SUM(CASE WHEN t.type = 'income' THEN t.amount ELSE 0 END), 0) as total_income,
            COALESCE(SUM(CASE WHEN t.type = 'expense' THEN t.amount ELSE 0 END), 0) as total_expenses
        %s %s`, baseQuery.String(), whereClause)

	err := r.db.QueryRow(ctx, sumQuery, args...).Scan(&stats.TotalIncome, &stats.TotalExpenses)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) { // ErrNoRows could mean 0 transactions match filter
		return nil, fmt.Errorf("failed to get total income/expenses: %w", err)
	}
	stats.Balance = stats.TotalIncome - stats.TotalExpenses

	// By Category (Income)
	incomeCategoryArgs := make([]interface{}, len(args))
	copy(incomeCategoryArgs, args)
	incomeCategoryWhereClause := whereClause
	incomeCategoryArgCount := argCount

	// Add type = 'income' if not already filtered by type, or if filtered by type='income'
	if filters.Type == nil || (filters.Type != nil && *filters.Type == model.TransactionTypeIncome) {
		if filters.Type == nil { // if no type filter add it
			if incomeCategoryWhereClause == "" {
				incomeCategoryWhereClause = fmt.Sprintf(" WHERE t.type = $%d", incomeCategoryArgCount)
			} else {
				incomeCategoryWhereClause += fmt.Sprintf(" AND t.type = $%d", incomeCategoryArgCount)
			}
			incomeCategoryArgs = append(incomeCategoryArgs, model.TransactionTypeIncome)
			//incomeCategoryArgCount++
		}
		categoryIncomeQuery := fmt.Sprintf(`SELECT t.category, COALESCE(SUM(t.amount), 0) %s %s GROUP BY t.category`, baseQuery.String(), incomeCategoryWhereClause)
		rows, err := r.db.Query(ctx, categoryIncomeQuery, incomeCategoryArgs...)
		if err != nil {
			return nil, fmt.Errorf("failed to get income by category: %w", err)
		}
		for rows.Next() {
			var category string
			var sum int64
			if err := rows.Scan(&category, &sum); err != nil {
				rows.Close()
				return nil, fmt.Errorf("failed to scan income by category: %w", err)
			}
			stats.ByCategoryIncome[category] = sum
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating income by category: %w", err)
		}
	}

	// By Category (Expense)
	expenseCategoryArgs := make([]interface{}, len(args))
	copy(expenseCategoryArgs, args)
	expenseCategoryWhereClause := whereClause
	expenseCategoryArgCount := argCount

	if filters.Type == nil || (filters.Type != nil && *filters.Type == model.TransactionTypeExpense) {
		if filters.Type == nil {
			if expenseCategoryWhereClause == "" {
				expenseCategoryWhereClause = fmt.Sprintf(" WHERE t.type = $%d", expenseCategoryArgCount)
			} else {
				expenseCategoryWhereClause += fmt.Sprintf(" AND t.type = $%d", expenseCategoryArgCount)
			}
			expenseCategoryArgs = append(expenseCategoryArgs, model.TransactionTypeExpense)
			//expenseCategoryArgCount++
		}
		categoryExpenseQuery := fmt.Sprintf(`SELECT t.category, COALESCE(SUM(t.amount), 0) %s %s GROUP BY t.category`, baseQuery.String(), expenseCategoryWhereClause)
		rows, err := r.db.Query(ctx, categoryExpenseQuery, expenseCategoryArgs...)
		if err != nil {
			return nil, fmt.Errorf("failed to get expense by category: %w", err)
		}
		for rows.Next() {
			var category string
			var sum int64
			if err := rows.Scan(&category, &sum); err != nil {
				rows.Close()
				return nil, fmt.Errorf("failed to scan expense by category: %w", err)
			}
			stats.ByCategoryExpense[category] = sum
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating expense by category: %w", err)
		}
	}

	// By User Spending

	userSpendingQuery := fmt.Sprintf(`
        SELECT 
            t.user_id, 
            u.phone,
            COALESCE(SUM(CASE WHEN t.type = 'expense' THEN t.amount ELSE 0 END), 0) as total_spent,
            COALESCE(SUM(CASE WHEN t.type = 'income' THEN t.amount ELSE 0 END), 0) as total_income,
            COUNT(t.id) as transaction_count
        %s %s GROUP BY t.user_id, u.phone`, baseQuery.String(), whereClause)

	rows, err := r.db.Query(ctx, userSpendingQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats by user: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var us model.UserStat
		if err := rows.Scan(&us.UserID, &us.UserPhone, &us.TotalSpent, &us.TotalIncome, &us.TransactionCount); err != nil {
			return nil, fmt.Errorf("failed to scan user stats: %w", err)
		}
		stats.ByUserSpending[us.UserID] = us
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user stats: %w", err)
	}

	return stats, nil
}
