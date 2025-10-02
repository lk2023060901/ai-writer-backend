package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Agent is the GORM model for agents table
type Agent struct {
	ID          string         `gorm:"primaryKey;type:varchar(36)"`
	Name        string         `gorm:"type:varchar(255);not null;index"`
	Description string         `gorm:"type:text"`
	Emoji       string         `gorm:"type:varchar(10)"`
	Prompt      string         `gorm:"type:text;not null"`
	Groups      StringArray    `gorm:"type:json"`
	Settings    JSON           `gorm:"type:json"`
	IsBuiltin   bool           `gorm:"default:false;index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// TableName specifies the table name
func (Agent) TableName() string {
	return "agents"
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
