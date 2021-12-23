package models

import "gorm.io/gorm"

func InitModels(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
		&UploadLimit{},
	)
}
