package models

import "time"

type User struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement"`
	Email        string `gorm:"index:idx_email,unique"`
	PasswordHash []byte
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	Admin        bool
}
