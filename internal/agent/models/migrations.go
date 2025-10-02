package models

import "gorm.io/gorm"

// AutoMigrate runs database migrations for agent domain
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Agent{},
	)
}
