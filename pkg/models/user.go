package models

import "time"

// TODO: make the ID the actual primary key
type User struct {
	ID           uint64
	Email        string `gorm:"primaryKey"`
	PasswordHash []byte
	CreatedAt    time.Time
}
