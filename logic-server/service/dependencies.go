package service

import (
	"context"
	"time"

	"geekedu/logic-server/model"
)

type CourseRepository interface {
	CreateCourse(course *model.Course) error
	ListCourses(page, pageSize int) ([]*model.Course, int64, error)
	GetCourseByID(id uint64) (*model.Course, error)
	UpdateCoverKey(id uint64, coverKey string) error
}

type VideoRepository interface {
	CreateVideo(video *model.CourseVideo) error
	GetVideosByCourseID(courseID uint64) ([]*model.CourseVideo, error)
	GetVideoByID(id uint64) (*model.CourseVideo, error)
}

type OrderRepository interface {
	CreateOrder(order *model.Order) error
	CheckPurchase(userID, courseID uint64) (bool, error)
}

type UserRepository interface {
	CreateUser(user *model.User) error
	GetUserByUsername(username string) (*model.User, error)
}

type Cache interface {
	Get(ctx context.Context, key string) (data []byte, hit bool, err error)
	Set(ctx context.Context, key string, value []byte, expiration time.Duration) error
	SetNX(ctx context.Context, key string, value string, expiration time.Duration) (bool, error)
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) error
}

type ObjectStorage interface {
	GenerateSignedURL(objectKey string, expireSec int64) (string, error)
	GeneratePresignedPutURL(objectKey string, expireSec int64) (string, error)
	GenerateCoverKey(courseID uint64, ext string) string
	GenerateVideoKey(courseID uint64, filename string) string
	InitiateMultipartUpload(objectKey string) (string, error)
	GeneratePresignedPartURL(objectKey, uploadID string, partNumber int) (string, error)
	CompleteMultipartUpload(objectKey, uploadID string, parts []UploadPart) error
}

type PasswordManager interface {
	Hash(password string) (string, error)
	Compare(hashedPassword, password string) error
}

type TokenProvider interface {
	Generate(userID int64, role int32) (string, error)
}

type UploadPart struct {
	PartNumber int
	ETag       string
}
