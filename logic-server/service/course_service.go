package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	"geekedu/common/errcode"
	pb "geekedu/common/pb"
	"geekedu/logic-server/model"

	"golang.org/x/sync/singleflight"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

type CourseServiceServer struct {
	pb.UnimplementedCourseServiceServer
	courseRepo CourseRepository
	videoRepo  VideoRepository
	cache      Cache
	storage    ObjectStorage

	courseListGroup singleflight.Group
}

type cachePrefixDeleter interface {
	DeletePrefix(ctx context.Context, prefix string) error
}

func NewCourseServiceServer(courseRepo CourseRepository, videoRepo VideoRepository, cache Cache, storage ObjectStorage) *CourseServiceServer {
	return &CourseServiceServer{
		courseRepo: courseRepo,
		videoRepo:  videoRepo,
		cache:      cache,
		storage:    storage,
	}
}

func (s *CourseServiceServer) CreateCourse(ctx context.Context, req *pb.CreateCourseRequest) (*pb.CreateCourseResponse, error) {
	if req.Title == "" || req.Price < 0 {
		return nil, errcode.ErrInvalidParams.ToGRPCError()
	}

	course := &model.Course{
		Title:       req.Title,
		Description: req.Description,
		Price:       req.Price,
		TeacherID:   uint64(req.TeacherId),
		CoverKey:    req.CoverKey,
	}
	if err := s.courseRepo.CreateCourse(course); err != nil {
		log.Printf("Failed to create course: %v", err)
		return nil, errcode.ErrInternal.ToGRPCError()
	}

	s.invalidateCourseListCache(ctx)
	return &pb.CreateCourseResponse{CourseId: int64(course.ID)}, nil
}

func (s *CourseServiceServer) GetCoverUploadURL(ctx context.Context, req *pb.GetCoverUploadURLRequest) (*pb.GetCoverUploadURLResponse, error) {
	if req.Filename == "" {
		return nil, errcode.ErrInvalidParams.ToGRPCError()
	}

	ext := strings.ToLower(filepath.Ext(req.Filename))
	if ext == "" {
		ext = ".jpg"
	}
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return nil, errcode.ErrInvalidParams.ToGRPCError()
	}

	objectKey := s.storage.GenerateCoverKey(0, ext)
	url, err := s.storage.GeneratePresignedPutURL(objectKey, 300)
	if err != nil {
		log.Printf("Failed to generate presigned PUT url: %v", err)
		return nil, errcode.ErrOSS.ToGRPCError()
	}

	return &pb.GetCoverUploadURLResponse{UploadUrl: url, CoverKey: objectKey}, nil
}

func (s *CourseServiceServer) ListCourses(ctx context.Context, req *pb.ListCoursesRequest) (*pb.ListCoursesResponse, error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	cacheKey := fmt.Sprintf("cache:courses:page:%d:size:%d", page, pageSize)

	//从缓存中获取课程列表
	if s.cache != nil {
		cachedData, hit, err := s.cache.Get(ctx, cacheKey)
		if err == nil && hit {
			var resp pb.ListCoursesResponse
			//反序列化缓存中的课程列表到pb.ListCoursesResponse
			if err := proto.Unmarshal(cachedData, &resp); err == nil {
				return &resp, nil
			}
			log.Printf("Failed to unmarshal cached course list: %v", err)
		} else if err != nil {
			log.Printf("Redis GET error: %v", err)
		}
	}

	//
	v, err, _ := s.courseListGroup.Do(cacheKey, func() (interface{}, error) {
		resp, err := s.loadCourseList(page, pageSize)
		if err != nil {
			return nil, err
		}
		if s.cache != nil {
			//序列化存储课程列表到缓存中
			data, err := proto.Marshal(resp)
			if err == nil {
				expiration := 5*time.Minute + time.Duration(rand.Intn(60))*time.Second
				//缓存课程列表，过期时间为5分钟到5分59随机值
				if err := s.cache.Set(context.Background(), cacheKey, data, expiration); err != nil {
					log.Printf("Failed to set course list cache: %v", err)
				}
			}
		}
		return resp, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*pb.ListCoursesResponse), nil
}

func (s *CourseServiceServer) GetCourse(ctx context.Context, req *pb.GetCourseRequest) (*pb.GetCourseResponse, error) {
	if req.CourseId <= 0 {
		return nil, errcode.ErrInvalidParams.ToGRPCError()
	}

	course, err := s.courseRepo.GetCourseByID(uint64(req.CourseId))
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Failed to get course %d: %v", req.CourseId, err)
			return nil, errcode.ErrInternal.ToGRPCError()
		}
		return nil, errcode.ErrCourseNotFound.ToGRPCError()
	}

	videos, err := s.videoRepo.GetVideosByCourseID(course.ID)
	if err != nil {
		log.Printf("Failed to get videos for course %d: %v", course.ID, err)
		return nil, errcode.ErrInternal.ToGRPCError()
	}

	return &pb.GetCourseResponse{
		Course: s.toCourseItem(course),
		Videos: toVideoItems(videos),
	}, nil
}

//加载课程列表
func (s *CourseServiceServer) loadCourseList(page, pageSize int) (*pb.ListCoursesResponse, error) {
	courses, total, err := s.courseRepo.ListCourses(page, pageSize)
	if err != nil {
		log.Printf("Failed to list courses: %v", err)
		return nil, errcode.ErrInternal.ToGRPCError()
	}

	items := make([]*pb.CourseItem, 0, len(courses))
	for _, course := range courses {
		items = append(items, s.toCourseItem(course))
	}
	return &pb.ListCoursesResponse{Courses: items, Total: total}, nil
}

//把model.Course转换为pb.CourseItem
func (s *CourseServiceServer) toCourseItem(course *model.Course) *pb.CourseItem {
	coverURL := ""
	if course.CoverKey != "" {
		url, err := s.storage.GenerateSignedURL(course.CoverKey, 3600)
		if err != nil {
			log.Printf("Failed to generate cover URL for course %d: %v", course.ID, err)
		} else {
			coverURL = url
		}
	}
	return &pb.CourseItem{
		Id:          int64(course.ID),
		Title:       course.Title,
		Description: course.Description,
		Price:       course.Price,
		CoverUrl:    coverURL,
		CreatedAt:   course.CreatedAt.Format("2006-01-02 15:04:05"),
		TeacherId:   int64(course.TeacherID),
	}
}

func toVideoItems(videos []*model.CourseVideo) []*pb.VideoItem {
	items := make([]*pb.VideoItem, 0, len(videos))
	for _, video := range videos {
		items = append(items, &pb.VideoItem{
			Id:        int64(video.ID),
			Title:     video.Title,
			SortOrder: int32(video.SortOrder),
			CourseId:  int64(video.CourseID),
		})
	}
	return items
}

func normalizePage(page, pageSize int32) (int, int) {
	p := int(page)
	s := int(pageSize)
	if p <= 0 {
		p = 1
	}
	if s <= 0 {
		s = 10
	}
	if s > 100 {
		s = 100
	}
	return p, s
}

func (s *CourseServiceServer) invalidateCourseListCache(ctx context.Context) {
	deleter, ok := s.cache.(cachePrefixDeleter)
	if !ok {
		return
	}
	if err := deleter.DeletePrefix(ctx, "cache:courses:"); err != nil {
		log.Printf("Failed to invalidate course list cache: %v", err)
	}
}
