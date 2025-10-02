package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Assistant is the GORM model for assistants table
type Assistant struct {
	ID     string `gorm:"primaryKey;type:varchar(36)"`
	UserID string `gorm:"type:varchar(36);not null;index"`

	Name   string      `gorm:"type:varchar(255);not null"`
	Emoji  string      `gorm:"type:varchar(10)"`
	Prompt string      `gorm:"type:text"`
	Type   string      `gorm:"type:varchar(50);default:'assistant'"`
	Tags   StringArray `gorm:"type:json"`

	KnowledgeBaseIDs StringArray `gorm:"type:json"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name
func (Assistant) TableName() string {
	return "assistants"
}

// Topic is the GORM model for topics table
type Topic struct {
	ID          string `gorm:"primaryKey;type:varchar(36)"`
	AssistantID string `gorm:"type:varchar(36);not null;index"`
	Name        string `gorm:"type:varchar(255);not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name
func (Topic) TableName() string {
	return "topics"
}

// StringArray is a custom type for string slice stored as JSON
type StringArray []string

// Scan implements sql.Scanner interface
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, s)
}

// Value implements driver.Valuer interface
func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// JSON is a custom type for JSON data
type JSON map[string]interface{}

// Scan implements sql.Scanner interface
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, j)
}

// Value implements driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}
