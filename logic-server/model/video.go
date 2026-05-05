package model

import "time"

type CourseVideo struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	CourseID  uint64    `gorm:"not null;index:idx_course_id" json:"course_id"`
	Title     string    `gorm:"type:varchar(256);not null" json:"title"`
	VideoKey  string    `gorm:"type:varchar(512);not null" json:"video_key"`
	Duration  uint32    `gorm:"not null;default:0" json:"duration"`
	SortOrder int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (CourseVideo) TableName() string {
	return "course_videos"
}
