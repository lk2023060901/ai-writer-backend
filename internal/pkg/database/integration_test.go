// +build integration

package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/lk2023060901/ai-writer-backend/internal/pkg/logger"
	"gorm.io/gorm"
)

// TestUser is a test model
type TestUser struct {
	ID        uint           `gorm:"primarykey"`
	Name      string         `gorm:"size:100;not null"`
	Email     string         `gorm:"size:255;uniqueIndex;not null"`
	Age       int            `gorm:"default:0"`
	Status    string         `gorm:"size:20;default:'active'"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (TestUser) TableName() string {
	return "test_users"
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) (*DB, func()) {
	// 从环境变量读取配置，如果没有则使用 docker-compose 默认值
	host := getEnv("TEST_DB_HOST", "localhost")
	port := 5432
	user := getEnv("TEST_DB_USER", "postgres")
	password := getEnv("TEST_DB_PASSWORD", "postgres")
	dbname := getEnv("TEST_DB_NAME", "aiwriter")

	cfg := &Config{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		DBName:   dbname,
		SSLMode:  "disable",

		MaxIdleConns:    5,
		MaxOpenConns:    10,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 10 * time.Minute,

		LogLevel:      "info",
		SlowThreshold: 200 * time.Millisecond,
		PrepareStmt:   true,
		AutoMigrate:   false,

		Timezone: "UTC",
	}

	log, err := logger.Development()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	db, err := New(cfg, log)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// 创建测试表
	if err := db.AutoMigrate(&TestUser{}); err != nil {
		t.Fatalf("Failed to migrate test table: %v", err)
	}

	// 清理函数
	cleanup := func() {
		// 删除测试表
		db.Exec("DROP TABLE IF EXISTS test_users")
		db.Close()
	}

	return db, cleanup
}

// getEnv gets environment variable or returns default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// TestDatabaseConnection tests database connection
func TestDatabaseConnection(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Test health check
	if err := db.HealthCheck(ctx); err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	// Test stats
	stats := db.Stats()
	if stats == nil {
		t.Error("Stats should not be nil")
	}

	t.Logf("Database stats: %+v", stats)
}

// TestCRUDOperations tests basic CRUD operations
func TestCRUDOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("Create", func(t *testing.T) {
		user := &TestUser{
			Name:   "John Doe",
			Email:  "john@example.com",
			Age:    30,
			Status: "active",
		}

		if err := db.WithContext(ctx).Create(user).Error; err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		if user.ID == 0 {
			t.Error("User ID should be set after creation")
		}

		t.Logf("Created user with ID: %d", user.ID)
	})

	t.Run("Read", func(t *testing.T) {
		var user TestUser
		if err := db.WithContext(ctx).First(&user, "email = ?", "john@example.com").Error; err != nil {
			t.Fatalf("Failed to read user: %v", err)
		}

		if user.Name != "John Doe" {
			t.Errorf("Expected name 'John Doe', got '%s'", user.Name)
		}

		t.Logf("Read user: %+v", user)
	})

	t.Run("Update", func(t *testing.T) {
		if err := db.WithContext(ctx).Model(&TestUser{}).
			Where("email = ?", "john@example.com").
			Update("age", 31).Error; err != nil {
			t.Fatalf("Failed to update user: %v", err)
		}

		var user TestUser
		db.WithContext(ctx).First(&user, "email = ?", "john@example.com")

		if user.Age != 31 {
			t.Errorf("Expected age 31, got %d", user.Age)
		}

		t.Logf("Updated user age to: %d", user.Age)
	})

	t.Run("Delete", func(t *testing.T) {
		if err := db.WithContext(ctx).Where("email = ?", "john@example.com").Delete(&TestUser{}).Error; err != nil {
			t.Fatalf("Failed to delete user: %v", err)
		}

		var count int64
		db.WithContext(ctx).Model(&TestUser{}).Where("email = ?", "john@example.com").Count(&count)

		if count != 0 {
			t.Errorf("Expected 0 users, got %d", count)
		}

		t.Log("Deleted user successfully")
	})
}

// TestTransactions tests transaction functionality
func TestTransactions(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("Successful Transaction", func(t *testing.T) {
		err := db.Transaction(ctx, func(ctx context.Context, tx *gorm.DB) error {
			user1 := &TestUser{Name: "Alice", Email: "alice@example.com", Age: 25}
			if err := tx.Create(user1).Error; err != nil {
				return err
			}

			user2 := &TestUser{Name: "Bob", Email: "bob@example.com", Age: 28}
			if err := tx.Create(user2).Error; err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			t.Fatalf("Transaction failed: %v", err)
		}

		// Verify users were created
		var count int64
		db.WithContext(ctx).Model(&TestUser{}).Count(&count)

		if count != 2 {
			t.Errorf("Expected 2 users, got %d", count)
		}

		t.Log("Transaction committed successfully")
	})

	t.Run("Failed Transaction with Rollback", func(t *testing.T) {
		// Clean up first
		db.Exec("DELETE FROM test_users")

		err := db.Transaction(ctx, func(ctx context.Context, tx *gorm.DB) error {
			user := &TestUser{Name: "Charlie", Email: "charlie@example.com", Age: 30}
			if err := tx.Create(user).Error; err != nil {
				return err
			}

			// Simulate error
			return gorm.ErrInvalidTransaction
		})

		if err == nil {
			t.Fatal("Transaction should have failed")
		}

		// Verify rollback
		var count int64
		db.WithContext(ctx).Model(&TestUser{}).Count(&count)

		if count != 0 {
			t.Errorf("Expected 0 users after rollback, got %d", count)
		}

		t.Log("Transaction rolled back successfully")
	})

	t.Run("Transaction Manager with Retry", func(t *testing.T) {
		// Clean up first
		db.Exec("DELETE FROM test_users")

		tm := NewTransactionManager(db)

		attempts := 0
		err := tm.ExecuteWithRetry(ctx, 3, func(ctx context.Context, tx *gorm.DB) error {
			attempts++
			if attempts < 2 {
				// Fail first attempt
				return gorm.ErrInvalidTransaction
			}

			user := &TestUser{Name: "David", Email: "david@example.com", Age: 35}
			return tx.Create(user).Error
		})

		if err != nil {
			t.Fatalf("Transaction with retry failed: %v", err)
		}

		if attempts != 2 {
			t.Errorf("Expected 2 attempts, got %d", attempts)
		}

		t.Logf("Transaction succeeded after %d attempts", attempts)
	})
}

// TestQueryHelpers tests query helper functions
func TestQueryHelpers(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Setup test data
	users := []TestUser{
		{Name: "User1", Email: "user1@example.com", Age: 20, Status: "active"},
		{Name: "User2", Email: "user2@example.com", Age: 25, Status: "active"},
		{Name: "User3", Email: "user3@example.com", Age: 30, Status: "inactive"},
		{Name: "User4", Email: "user4@example.com", Age: 35, Status: "active"},
		{Name: "User5", Email: "user5@example.com", Age: 40, Status: "active"},
	}

	if err := db.WithContext(ctx).Create(&users).Error; err != nil {
		t.Fatalf("Failed to create test users: %v", err)
	}

	t.Run("Pagination", func(t *testing.T) {
		var result []TestUser
		err := db.WithContext(ctx).
			Scopes(Paginate(1, 2)).
			Find(&result).Error

		if err != nil {
			t.Fatalf("Pagination failed: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("Expected 2 users, got %d", len(result))
		}

		t.Logf("Paginated results: %d users", len(result))
	})

	t.Run("FindWithPagination", func(t *testing.T) {
		var users []TestUser
		result, err := FindWithPagination(ctx, db.DB.Model(&TestUser{}), &users, 1, 3)

		if err != nil {
			t.Fatalf("FindWithPagination failed: %v", err)
		}

		if result.Total != 5 {
			t.Errorf("Expected total 5, got %d", result.Total)
		}

		if len(users) != 3 {
			t.Errorf("Expected 3 users, got %d", len(users))
		}

		if result.TotalPages != 2 {
			t.Errorf("Expected 2 pages, got %d", result.TotalPages)
		}

		t.Logf("Pagination result: %+v", result)
	})

	t.Run("OrderBy", func(t *testing.T) {
		var result []TestUser
		err := db.WithContext(ctx).
			Scopes(OrderBy("age", true)).
			Find(&result).Error

		if err != nil {
			t.Fatalf("OrderBy failed: %v", err)
		}

		if result[0].Age != 40 {
			t.Errorf("Expected first user age 40, got %d", result[0].Age)
		}

		t.Log("OrderBy DESC successful")
	})

	t.Run("WhereIf", func(t *testing.T) {
		status := "active"
		var result []TestUser

		err := db.WithContext(ctx).
			Scopes(WhereIf(status != "", "status = ?", status)).
			Find(&result).Error

		if err != nil {
			t.Fatalf("WhereIf failed: %v", err)
		}

		if len(result) != 4 {
			t.Errorf("Expected 4 active users, got %d", len(result))
		}

		t.Log("WhereIf successful")
	})

	t.Run("Exists", func(t *testing.T) {
		exists, err := Exists(ctx, db.DB, &TestUser{}, "email = ?", "user1@example.com")

		if err != nil {
			t.Fatalf("Exists failed: %v", err)
		}

		if !exists {
			t.Error("Expected user to exist")
		}

		t.Log("Exists check successful")
	})

	t.Run("Count", func(t *testing.T) {
		count, err := Count(ctx, db.DB, &TestUser{}, "status = ?", "active")

		if err != nil {
			t.Fatalf("Count failed: %v", err)
		}

		if count != 4 {
			t.Errorf("Expected count 4, got %d", count)
		}

		t.Logf("Count result: %d", count)
	})

	t.Run("QueryBuilder", func(t *testing.T) {
		qb := NewQueryBuilder(db.DB).
			Where("status = ?", "active").
			Where("age > ?", 25).
			Order("age ASC").
			Limit(2)

		var result []TestUser
		if err := qb.Find(ctx, &result); err != nil {
			t.Fatalf("QueryBuilder Find failed: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("Expected 2 users, got %d", len(result))
		}

		count, err := qb.Count(ctx)
		if err != nil {
			t.Fatalf("QueryBuilder Count failed: %v", err)
		}

		if count != 2 {
			t.Errorf("Expected count 2, got %d", count)
		}

		t.Log("QueryBuilder successful")
	})
}

// TestBatchOperations tests batch operations
func TestBatchOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("BatchInsert", func(t *testing.T) {
		users := []TestUser{}
		for i := 0; i < 50; i++ {
			users = append(users, TestUser{
				Name:   "BatchUser" + string(rune(i)),
				Email:  "batch" + string(rune(i)) + "@example.com",
				Age:    20 + i,
				Status: "active",
			})
		}

		if err := BatchInsert(ctx, db.DB, users, 10); err != nil {
			t.Fatalf("BatchInsert failed: %v", err)
		}

		var count int64
		db.WithContext(ctx).Model(&TestUser{}).Count(&count)

		if count != 50 {
			t.Errorf("Expected 50 users, got %d", count)
		}

		t.Log("BatchInsert successful")
	})

	t.Run("BulkUpdate", func(t *testing.T) {
		updates := map[string]interface{}{
			"status": "inactive",
		}

		if err := BulkUpdate(ctx, db.DB, &TestUser{}, updates, "age > ?", 50); err != nil {
			t.Fatalf("BulkUpdate failed: %v", err)
		}

		var count int64
		db.WithContext(ctx).Model(&TestUser{}).Where("status = ?", "inactive").Count(&count)

		if count == 0 {
			t.Error("Expected some users to be updated")
		}

		t.Logf("BulkUpdate successful, updated %d users", count)
	})
}

// TestSoftDelete tests soft delete functionality
func TestSoftDelete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	user := &TestUser{
		Name:   "ToDelete",
		Email:  "todelete@example.com",
		Age:    30,
		Status: "active",
	}

	db.WithContext(ctx).Create(user)

	t.Run("SoftDelete", func(t *testing.T) {
		if err := SoftDelete(ctx, db.DB, &TestUser{}, user.ID); err != nil {
			t.Fatalf("SoftDelete failed: %v", err)
		}

		// Should not find with normal query
		var found TestUser
		err := db.WithContext(ctx).First(&found, user.ID).Error

		if !IsRecordNotFoundError(err) {
			t.Error("Expected record not found error")
		}

		t.Log("SoftDelete successful")
	})

	t.Run("Restore", func(t *testing.T) {
		if err := Restore(ctx, db.DB, &TestUser{}, user.ID); err != nil {
			t.Fatalf("Restore failed: %v", err)
		}

		// Should find after restore
		var found TestUser
		if err := db.WithContext(ctx).First(&found, user.ID).Error; err != nil {
			t.Fatalf("Failed to find restored user: %v", err)
		}

		t.Log("Restore successful")
	})

	t.Run("HardDelete", func(t *testing.T) {
		if err := HardDelete(ctx, db.DB, &TestUser{}, user.ID); err != nil {
			t.Fatalf("HardDelete failed: %v", err)
		}

		// Should not find even with Unscoped
		var found TestUser
		err := db.WithContext(ctx).Unscoped().First(&found, user.ID).Error

		if !IsRecordNotFoundError(err) {
			t.Error("Expected record not found error after hard delete")
		}

		t.Log("HardDelete successful")
	})
}
