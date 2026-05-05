package model

import "time"

type Order struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint64    `gorm:"not null;index:idx_user_id" json:"user_id"`
	CourseID  uint64    `gorm:"not null;index:idx_course_id" json:"course_id"`
	Price     float64   `gorm:"type:decimal(10,2);not null;default:0.00" json:"price"`
	Status    int8      `gorm:"type:tinyint;not null;default:1" json:"status"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (Order) TableName() string {
	return "orders"
}
