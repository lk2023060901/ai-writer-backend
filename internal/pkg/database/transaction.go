package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TxFunc defines a transaction function
type TxFunc func(ctx context.Context, tx *gorm.DB) error

// Transaction executes a function within a database transaction
func (db *DB) Transaction(ctx context.Context, fn TxFunc) error {
	return db.TransactionWithOptions(ctx, &sql.TxOptions{}, fn)
}

// TransactionWithOptions executes a function within a database transaction with custom options
func (db *DB) TransactionWithOptions(ctx context.Context, opts *sql.TxOptions, fn TxFunc) error {
	db.logger.WithContext(ctx).Debug("starting database transaction")

	return db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := fn(ctx, tx); err != nil {
			db.logger.WithContext(ctx).Error("transaction failed, rolling back",
				zap.Error(err),
			)
			return err
		}

		db.logger.WithContext(ctx).Debug("transaction committed successfully")
		return nil
	}, opts)
}

// BeginTx starts a new transaction
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) *gorm.DB {
	db.logger.WithContext(ctx).Debug("beginning transaction")
	return db.DB.WithContext(ctx).Begin(opts)
}

// Commit commits the transaction
func (db *DB) Commit(tx *gorm.DB) error {
	if tx == nil {
		return errors.New("transaction is nil")
	}

	if err := tx.Commit().Error; err != nil {
		db.logger.Error("failed to commit transaction", zap.Error(err))
		return fmt.Errorf("commit failed: %w", err)
	}

	db.logger.Debug("transaction committed")
	return nil
}

// Rollback rolls back the transaction
func (db *DB) Rollback(tx *gorm.DB) error {
	if tx == nil {
		return errors.New("transaction is nil")
	}

	if err := tx.Rollback().Error; err != nil {
		db.logger.Error("failed to rollback transaction", zap.Error(err))
		return fmt.Errorf("rollback failed: %w", err)
	}

	db.logger.Debug("transaction rolled back")
	return nil
}

// TransactionManager provides transaction management utilities
type TransactionManager struct {
	db *DB
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(db *DB) *TransactionManager {
	return &TransactionManager{db: db}
}

// Execute executes a function within a transaction with automatic retry
func (tm *TransactionManager) Execute(ctx context.Context, fn TxFunc) error {
	return tm.ExecuteWithRetry(ctx, 3, fn)
}

// ExecuteWithRetry executes a function within a transaction with retry on specific errors
func (tm *TransactionManager) ExecuteWithRetry(ctx context.Context, maxRetries int, fn TxFunc) error {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			tm.db.logger.WithContext(ctx).Warn("retrying transaction",
				zap.Int("attempt", i+1),
				zap.Int("max_retries", maxRetries),
				zap.Error(lastErr),
			)
		}

		err := tm.db.Transaction(ctx, fn)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return err
		}
	}

	return fmt.Errorf("transaction failed after %d retries: %w", maxRetries, lastErr)
}

// ExecuteNested executes nested transactions using savepoints
func (tm *TransactionManager) ExecuteNested(ctx context.Context, tx *gorm.DB, fn TxFunc) error {
	// Use SavePoint for nested transactions
	savepointName := "sp_nested"

	// Create savepoint
	if err := tx.SavePoint(savepointName).Error; err != nil {
		return fmt.Errorf("failed to create savepoint: %w", err)
	}

	// Execute function
	if err := fn(ctx, tx); err != nil {
		// Rollback to savepoint on error
		if rbErr := tx.RollbackTo(savepointName).Error; rbErr != nil {
			return fmt.Errorf("failed to rollback to savepoint: %w (original error: %v)", rbErr, err)
		}
		return err
	}

	return nil
}

// ReadCommitted executes a function in READ COMMITTED isolation level
func (tm *TransactionManager) ReadCommitted(ctx context.Context, fn TxFunc) error {
	return tm.db.TransactionWithOptions(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
	}, fn)
}

// ReadUncommitted executes a function in READ UNCOMMITTED isolation level
func (tm *TransactionManager) ReadUncommitted(ctx context.Context, fn TxFunc) error {
	return tm.db.TransactionWithOptions(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadUncommitted,
		ReadOnly:  false,
	}, fn)
}

// RepeatableRead executes a function in REPEATABLE READ isolation level
func (tm *TransactionManager) RepeatableRead(ctx context.Context, fn TxFunc) error {
	return tm.db.TransactionWithOptions(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  false,
	}, fn)
}

// Serializable executes a function in SERIALIZABLE isolation level
func (tm *TransactionManager) Serializable(ctx context.Context, fn TxFunc) error {
	return tm.db.TransactionWithOptions(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	}, fn)
}

// ReadOnly executes a read-only transaction
func (tm *TransactionManager) ReadOnly(ctx context.Context, fn TxFunc) error {
	return tm.db.TransactionWithOptions(ctx, &sql.TxOptions{
		ReadOnly: true,
	}, fn)
}

// isRetryableError checks if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// PostgreSQL serialization failure error code: 40001
	// PostgreSQL deadlock detected error code: 40P01
	errMsg := err.Error()

	return errMsg == "ERROR: could not serialize access due to concurrent update (SQLSTATE 40001)" ||
		errMsg == "ERROR: deadlock detected (SQLSTATE 40P01)" ||
		errMsg == "ERROR: could not serialize access due to read/write dependencies among transactions (SQLSTATE 40001)"
}

// TransactionKey is the context key for storing transaction
type TransactionKey struct{}

// ContextWithTransaction adds transaction to context
func ContextWithTransaction(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, TransactionKey{}, tx)
}

// TransactionFromContext extracts transaction from context
func TransactionFromContext(ctx context.Context) (*gorm.DB, bool) {
	tx, ok := ctx.Value(TransactionKey{}).(*gorm.DB)
	return tx, ok
}

// GetDBFromContext returns the database instance from context if transaction exists, otherwise returns the original DB
func (db *DB) GetDBFromContext(ctx context.Context) *gorm.DB {
	if tx, ok := TransactionFromContext(ctx); ok {
		return tx
	}
	return db.DB.WithContext(ctx)
}
