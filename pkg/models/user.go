package models

import "time"

type User struct {
	Email        string `gorm:"primaryKey"`
	PasswordHash []byte
	CreatedAt    time.Time
}
