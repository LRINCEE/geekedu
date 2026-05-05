package dao

import (
	"geekedu/logic-server/model"

	"gorm.io/gorm"
)

type VideoDao struct {
	db *gorm.DB
}

func NewVideoDao() *VideoDao {
	return &VideoDao{db: GetDB()}
}

func NewVideoDaoWithDB(db *gorm.DB) *VideoDao {
	return &VideoDao{db: db}
}

func (d *VideoDao) CreateVideo(video *model.CourseVideo) error {
	return d.db.Create(video).Error
}

func (d *VideoDao) GetVideosByCourseID(courseID uint64) ([]*model.CourseVideo, error) {
	var videos []*model.CourseVideo
	// 只查需要的列，避免拉取长文本字段(VideoKey 等)
	err := d.db.Select("id", "title", "sort_order", "course_id").
		Where("course_id = ?", courseID).
		Order("sort_order ASC").
		Find(&videos).Error
	return videos, err
}

func (d *VideoDao) GetVideoByID(id uint64) (*model.CourseVideo, error) {
	var video model.CourseVideo
	err := d.db.Where("id = ?", id).First(&video).Error
	if err != nil {
		return nil, err
	}
	return &video, nil
}
