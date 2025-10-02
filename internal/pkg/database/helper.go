package database

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// Paginate adds pagination to a query
func Paginate(page, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page < 1 {
			page = 1
		}
		if pageSize < 1 {
			pageSize = 10
		}
		if pageSize > 100 {
			pageSize = 100 // Max page size
		}

		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

// OrderBy adds ordering to a query
func OrderBy(field string, desc bool) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		order := field
		if desc {
			order = field + " DESC"
		}
		return db.Order(order)
	}
}

// WhereIf conditionally adds a where clause
func WhereIf(condition bool, query interface{}, args ...interface{}) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if condition {
			return db.Where(query, args...)
		}
		return db
	}
}

// Preloads preloads multiple associations
func Preloads(relations ...string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		for _, relation := range relations {
			db = db.Preload(relation)
		}
		return db
	}
}

// Select adds select fields
func Select(fields ...string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Select(fields)
	}
}

// Distinct adds distinct clause
func Distinct(columns ...string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(columns) == 0 {
			return db.Distinct()
		}
		// Convert []string to []interface{}
		args := make([]interface{}, len(columns))
		for i, col := range columns {
			args[i] = col
		}
		return db.Distinct(args...)
	}
}

// PageResult represents a paginated result
type PageResult struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// FindWithPagination finds records with pagination
func FindWithPagination(ctx context.Context, db *gorm.DB, dest interface{}, page, pageSize int) (*PageResult, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var total int64

	// Count total records
	if err := db.WithContext(ctx).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count records: %w", err)
	}

	// Calculate total pages
	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	// Find records with pagination
	offset := (page - 1) * pageSize
	if err := db.WithContext(ctx).Offset(offset).Limit(pageSize).Find(dest).Error; err != nil {
		return nil, fmt.Errorf("failed to find records: %w", err)
	}

	return &PageResult{
		Data:       dest,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// Exists checks if a record exists
func Exists(ctx context.Context, db *gorm.DB, model interface{}, query interface{}, args ...interface{}) (bool, error) {
	var count int64
	err := db.WithContext(ctx).Model(model).Where(query, args...).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// FirstOrCreate finds the first record or creates a new one
func FirstOrCreate(ctx context.Context, db *gorm.DB, dest interface{}, conds ...interface{}) error {
	return db.WithContext(ctx).FirstOrCreate(dest, conds...).Error
}

// UpdateFields updates specific fields
func UpdateFields(ctx context.Context, db *gorm.DB, model interface{}, updates map[string]interface{}) error {
	return db.WithContext(ctx).Model(model).Updates(updates).Error
}

// BatchInsert inserts records in batches
func BatchInsert(ctx context.Context, db *gorm.DB, records interface{}, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 100
	}
	return db.WithContext(ctx).CreateInBatches(records, batchSize).Error
}

// SoftDelete performs soft delete
func SoftDelete(ctx context.Context, db *gorm.DB, model interface{}, id interface{}) error {
	return db.WithContext(ctx).Delete(model, id).Error
}

// HardDelete performs hard delete (permanent)
func HardDelete(ctx context.Context, db *gorm.DB, model interface{}, id interface{}) error {
	return db.WithContext(ctx).Unscoped().Delete(model, id).Error
}

// Restore restores a soft deleted record
func Restore(ctx context.Context, db *gorm.DB, model interface{}, id interface{}) error {
	return db.WithContext(ctx).Model(model).Unscoped().Where("id = ?", id).Update("deleted_at", nil).Error
}

// BulkUpdate updates multiple records
func BulkUpdate(ctx context.Context, db *gorm.DB, model interface{}, updates map[string]interface{}, query interface{}, args ...interface{}) error {
	return db.WithContext(ctx).Model(model).Where(query, args...).Updates(updates).Error
}

// FindInBatches processes records in batches
func FindInBatches(ctx context.Context, db *gorm.DB, dest interface{}, batchSize int, fn func(tx *gorm.DB, batch int) error) error {
	return db.WithContext(ctx).FindInBatches(dest, batchSize, fn).Error
}

// Pluck gets a single column as a slice
func Pluck(ctx context.Context, db *gorm.DB, column string, dest interface{}) error {
	return db.WithContext(ctx).Pluck(column, dest).Error
}

// Count counts records
func Count(ctx context.Context, db *gorm.DB, model interface{}, query interface{}, args ...interface{}) (int64, error) {
	var count int64
	err := db.WithContext(ctx).Model(model).Where(query, args...).Count(&count).Error
	return count, err
}

// QueryBuilder helps build complex queries
type QueryBuilder struct {
	db     *gorm.DB
	scopes []func(*gorm.DB) *gorm.DB
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder(db *gorm.DB) *QueryBuilder {
	return &QueryBuilder{
		db:     db,
		scopes: make([]func(*gorm.DB) *gorm.DB, 0),
	}
}

// Where adds a where condition
func (qb *QueryBuilder) Where(query interface{}, args ...interface{}) *QueryBuilder {
	qb.scopes = append(qb.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Where(query, args...)
	})
	return qb
}

// Or adds an or condition
func (qb *QueryBuilder) Or(query interface{}, args ...interface{}) *QueryBuilder {
	qb.scopes = append(qb.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Or(query, args...)
	})
	return qb
}

// Order adds ordering
func (qb *QueryBuilder) Order(value interface{}) *QueryBuilder {
	qb.scopes = append(qb.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Order(value)
	})
	return qb
}

// Limit adds limit
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.scopes = append(qb.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Limit(limit)
	})
	return qb
}

// Offset adds offset
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.scopes = append(qb.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Offset(offset)
	})
	return qb
}

// Preload adds preload
func (qb *QueryBuilder) Preload(query string, args ...interface{}) *QueryBuilder {
	qb.scopes = append(qb.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Preload(query, args...)
	})
	return qb
}

// Select adds select fields
func (qb *QueryBuilder) Select(query interface{}, args ...interface{}) *QueryBuilder {
	qb.scopes = append(qb.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Select(query, args...)
	})
	return qb
}

// Group adds group by
func (qb *QueryBuilder) Group(name string) *QueryBuilder {
	qb.scopes = append(qb.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Group(name)
	})
	return qb
}

// Having adds having clause
func (qb *QueryBuilder) Having(query interface{}, args ...interface{}) *QueryBuilder {
	qb.scopes = append(qb.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Having(query, args...)
	})
	return qb
}

// Join adds join clause
func (qb *QueryBuilder) Join(query string, args ...interface{}) *QueryBuilder {
	qb.scopes = append(qb.scopes, func(db *gorm.DB) *gorm.DB {
		return db.Joins(query, args...)
	})
	return qb
}

// Build returns the final query
func (qb *QueryBuilder) Build() *gorm.DB {
	return qb.db.Scopes(qb.scopes...)
}

// Find executes find query
func (qb *QueryBuilder) Find(ctx context.Context, dest interface{}) error {
	return qb.Build().WithContext(ctx).Find(dest).Error
}

// First executes first query
func (qb *QueryBuilder) First(ctx context.Context, dest interface{}) error {
	return qb.Build().WithContext(ctx).First(dest).Error
}

// Count executes count query
func (qb *QueryBuilder) Count(ctx context.Context) (int64, error) {
	var count int64
	err := qb.Build().WithContext(ctx).Count(&count).Error
	return count, err
}
