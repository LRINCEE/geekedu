package model

import "time"

type Course struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Title       string    `gorm:"type:varchar(256);not null" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	CoverKey    string    `gorm:"type:varchar(512);not null;default:''" json:"cover_key"`
	Price       float64   `gorm:"type:decimal(10,2);not null;default:0.00" json:"price"`
	TeacherID   uint64    `gorm:"not null;index:idx_teacher_id" json:"teacher_id"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Course) TableName() string {
	return "courses"
}
