package model

import "time"

type User struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"type:varchar(64);uniqueIndex:idx_username;not null" json:"username"`
	Password  string    `gorm:"type:varchar(256);not null" json:"-"`
	Role      int8      `gorm:"type:tinyint;not null;default:0" json:"role"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}
