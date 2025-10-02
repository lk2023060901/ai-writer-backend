package models

import "gorm.io/gorm"

// AutoMigrate runs database migrations for assistant domain
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Assistant{},
		&Topic{},
	)
}
