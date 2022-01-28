package models

type UploadLimit struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	UserID    uint64 `gorm:"index:upload_limit_user_id_idx,unique"`
	User      *User
	Unlimited bool
	RateLimit int64
	Burst     int64
}

type DownloadLimit struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	UserID    uint64 `gorm:"index:download_limit_user_id_idx,unique"`
	User      *User
	Unlimited bool
	RateLimit int64
	Burst     int64
}
