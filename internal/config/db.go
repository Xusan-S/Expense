package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DBConfig holds database connection parameters
type DBConfig struct {
	DSN string
}

// LoadDBConfig loads database configuration from environment variables
func LoadDBConfig() (*DBConfig, error) {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	if dbHost == "" || dbPort == "" || dbUser == "" || dbName == "" {
		return nil, fmt.Errorf("database environment variables not set (DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME)")
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	return &DBConfig{DSN: dsn}, nil
}

// ConnectDB establishes a connection to the PostgreSQL database
func ConnectDB(cfg *DBConfig) (*pgxpool.Pool, error) {
	var pool *pgxpool.Pool
	var err error

	// Retry connecting to the database a few times
	maxRetries := 5
	retryInterval := 5 * time.Second

	for i := 0; i < maxRetries; i++ {
		pool, err = pgxpool.New(context.Background(), cfg.DSN)
		if err == nil {
			// Try to ping the database
			err = pool.Ping(context.Background())
			if err == nil {
				log.Println("Successfully connected to PostgreSQL!")
				return pool, nil
			}
		}
		log.Printf("Failed to connect to database (attempt %d/%d): %v. Retrying in %v...", i+1, maxRetries, err, retryInterval)
		time.Sleep(retryInterval)
	}
	return nil, fmt.Errorf("unable to connect to database after %d attempts: %w", maxRetries, err)
}

// AutoMigrate creates tables if they don't exist
func AutoMigrate(db *pgxpool.Pool) error {
	sql := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		phone TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL CHECK (role IN ('user', 'admin')) DEFAULT 'user',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS transactions (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT NOT NULL,
		amount BIGINT NOT NULL, -- in smallest currency unit (e.g., cents)
		type VARCHAR(50) NOT NULL CHECK (type IN ('income', 'expense')),
		category VARCHAR(100) NOT NULL,
		description TEXT,
		transaction_date TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
		receipt_path TEXT, -- stores relative path to the uploaded file
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	-- Indexes for performance
	CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
	CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(type);
	CREATE INDEX IF NOT EXISTS idx_transactions_category ON transactions(category);
	CREATE INDEX IF NOT EXISTS idx_transactions_transaction_date ON transactions(transaction_date);

    -- Function to update updated_at column
    CREATE OR REPLACE FUNCTION update_updated_at_column()
    RETURNS TRIGGER AS $$
    BEGIN
       NEW.updated_at = NOW();
       RETURN NEW;
    END;
    $$ language 'plpgsql';

    -- Trigger for transactions table
    DO $$
    BEGIN
        IF NOT EXISTS (
            SELECT 1
            FROM pg_trigger
            WHERE tgname = 'set_transactions_updated_at' AND tgrelid = 'transactions'::regclass
        ) THEN
            CREATE TRIGGER set_transactions_updated_at
            BEFORE UPDATE ON transactions
            FOR EACH ROW
            EXECUTE FUNCTION update_updated_at_column();
        END IF;
    END
    $$;
	`
	_, err := db.Exec(context.Background(), sql)
	if err != nil {
		return fmt.Errorf("unable to apply migrations: %w", err)
	}

	log.Println("AutoMigrate applied successfully")
	return nil
}