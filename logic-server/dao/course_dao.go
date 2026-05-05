package dao

import (
	"geekedu/logic-server/model"

	"gorm.io/gorm"
)

type CourseDao struct {
	db *gorm.DB
}

func NewCourseDao() *CourseDao {
	return &CourseDao{db: GetDB()}
}

func NewCourseDaoWithDB(db *gorm.DB) *CourseDao {
	return &CourseDao{db: db}
}

func (d *CourseDao) CreateCourse(course *model.Course) error {
	return d.db.Create(course).Error
}

func (d *CourseDao) ListCourses(page, pageSize int) ([]*model.Course, int64, error) {
	var courses []*model.Course
	var total int64

	d.db.Model(&model.Course{}).Count(&total)

	offset := (page - 1) * pageSize
	// 避免 SELECT *，仅查询需要的字段，减少网络传输和内存开销
	err := d.db.Select("id", "title", "description", "price", "cover_key", "teacher_id", "created_at").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&courses).Error
	return courses, total, err
}

func (d *CourseDao) GetCourseByID(id uint64) (*model.Course, error) {
	var course model.Course
	// 显式指定需要的列，避免 SELECT * 带来的性能损耗
	err := d.db.Select("id", "title", "description", "cover_key", "price", "teacher_id", "created_at").
		Where("id = ?", id).
		First(&course).Error
	if err != nil {
		return nil, err
	}
	return &course, nil
}

func (d *CourseDao) UpdateCoverKey(id uint64, coverKey string) error {
	return d.db.Model(&model.Course{}).Where("id = ?", id).Update("cover_key", coverKey).Error
}
