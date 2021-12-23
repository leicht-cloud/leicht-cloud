package models

type UploadLimit struct {
	ID        int64 `gorm:"primaryKey;autoIncrement"`
	UserID    uint64
	User      *User
	Unlimited bool
	RateLimit float64
	Burst     int64
}
