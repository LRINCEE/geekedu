// logic-server/service/course_service_test.go

package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"geekedu/common/pb"
	"geekedu/logic-server/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// ======== Mock 定义 ========

type mockCourseRepoForCourse struct {
	courses []*model.Course
	total   int64
	err     error
}

func (m *mockCourseRepoForCourse) CreateCourse(course *model.Course) error {
	if m.err != nil {
		return m.err
	}
	course.ID = uint64(len(m.courses) + 1)
	m.courses = append(m.courses, course)
	return nil
}

func (m *mockCourseRepoForCourse) ListCourses(page, pageSize int) ([]*model.Course, int64, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.courses, m.total, nil
}

func (m *mockCourseRepoForCourse) GetCourseByID(id uint64) (*model.Course, error) {
	for _, c := range m.courses {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockCourseRepoForCourse) UpdateCoverKey(id uint64, coverKey string) error { return nil }

type mockVideoRepo struct{}

func (m *mockVideoRepo) CreateVideo(video *model.CourseVideo) error { return nil }
func (m *mockVideoRepo) GetVideosByCourseID(courseID uint64) ([]*model.CourseVideo, error) {
	return []*model.CourseVideo{}, nil
}
func (m *mockVideoRepo) GetVideoByID(id uint64) (*model.CourseVideo, error) {
	return nil, gorm.ErrRecordNotFound
}

type mockObjectStorage struct {
	err error
}

func (m *mockObjectStorage) GenerateSignedURL(key string, exp int64) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "https://mock-signed-url.com/" + key + "?sig=xxx&expires=3600", nil
}
func (m *mockObjectStorage) GeneratePresignedPutURL(k string, e int64) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "https://mock-put.com", nil
}
func (m *mockObjectStorage) GenerateCoverKey(id uint64, ext string) string    { return "" }
func (m *mockObjectStorage) GenerateVideoKey(id uint64, f string) string      { return "" }
func (m *mockObjectStorage) InitiateMultipartUpload(k string) (string, error) { return "", nil }
func (m *mockObjectStorage) GeneratePresignedPartURL(k, u string, n int) (string, error) {
	return "", nil
}
func (m *mockObjectStorage) CompleteMultipartUpload(k, u string, p []UploadPart) error { return nil }

// ======== 测试用例 ========

func TestCourseService_ListCourses(t *testing.T) {

	sampleCourses := []*model.Course{
		{ID: 1, Title: "Go入门", Price: 0, CoverKey: "cover1.jpg"},
		{ID: 2, Title: "gRPC深入", Price: 99, CoverKey: "cover2.jpg"},
	}

	tests := []struct {
		name         string
		page, size   int
		cache        *mockCache // 可以为 nil 表示无缓存
		courseRepo   *mockCourseRepoForCourse
		wantErr      bool
		wantGRPCCode codes.Code
		wantCount    int
		wantTotal    int64
		expectCache  bool // 期望结果是否写入缓存
	}{
		{
			name: "缓存命中直接返回",
			page: 1, size: 10,
			cache: func() *mockCache {
				// 预先在缓存中放入数据
				cachedResp := &pb.ListCoursesResponse{
					Courses: []*pb.CourseItem{{Id: 1, Title: "from-cache"}},
					Total:   99,
				}
				data, _ := proto.Marshal(cachedResp)
				c := newMockCache()
				c.locks["cache:courses:page:1:size:10"] = string(data)
				return c
			}(),
			courseRepo:  &mockCourseRepoForCourse{}, // 不应该被调用
			wantCount:   1,                          // 缓存里的数量
			wantTotal:   99,                         // 缓存里的 total
			expectCache: false,                      // 不会回写缓存（因为是命中）
		},
		{
			name: "缓存未命中查库并回写",
			page: 1, size: 10,
			cache:       newMockCache(), // 空缓存
			courseRepo:  &mockCourseRepoForCourse{courses: sampleCourses, total: 2},
			wantCount:   2,
			wantTotal:   2,
			expectCache: true,
		},
		{
			name: "课程不存在返回空列表",
			page: 1, size: 10,
			cache:       newMockCache(),
			courseRepo:  &mockCourseRepoForCourse{courses: []*model.Course{}, total: 0},
			wantCount:   0,
			wantTotal:   0,
			expectCache: false, // 返回空列表时不应缓存
		},
		{
			name: "page=0 应默认为1",
			page: 0, size: 10,
			cache:      newMockCache(),
			courseRepo: &mockCourseRepoForCourse{courses: sampleCourses, total: 2},
			wantCount:  2,
			wantTotal:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			videoRepo := &mockVideoRepo{}
			storage := &mockObjectStorage{}

			svc := NewCourseServiceServer(tt.courseRepo, videoRepo, tt.cache, storage)

			req := &pb.ListCoursesRequest{
				Page:     int32(tt.page),
				PageSize: int32(tt.size),
			}

			resp, err := svc.ListCourses(context.Background(), req)

			if tt.wantErr {
				require.Error(t, err)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantGRPCCode, st.Code())
			} else {
				require.NoError(t, err)
				assert.Len(t, resp.Courses, tt.wantCount)
				assert.Equal(t, tt.wantTotal, resp.Total)

				if tt.expectCache && tt.cache != nil {
					// 验证缓存确实被写入了
					cached, hit, _ := tt.cache.Get(context.Background(),
						"cache:courses:page:1:size:10")
					assert.True(t, hit, "缓存应该被回写")
					assert.NotEmpty(t, cached, "缓存数据不应为空")

					// 反序列化验证数据一致性
					var cachedResp pb.ListCoursesResponse
					err := proto.Unmarshal(cached, &cachedResp)
					assert.NoError(t, err)
					assert.Equal(t, tt.wantTotal, cachedResp.Total)
				}
			}
		})
	}
}

// ★ singleflight 行为测试：并发请求只执行一次查库
func TestCourseService_SingleFlightPreventsDBPenetration(t *testing.T) {
	callCount := 0 // 记录 ListCourses 被调用的次数

	courseRepo := &mockCourseRepoForCourse{
		courses: []*model.Course{
			{ID: 1, Title: "test", Price: 0},
		},
		total: 1,
		// 用包装函数记录调用次数
	}

	// 包装 ListCourses 来计数
	originalList := courseRepo.ListCourses
	originalList = func(page, pageSize int) ([]*model.Course, int64, error) {
		callCount++
		time.Sleep(50 * time.Millisecond) // 模拟慢查询
		return originalList(page, pageSize)
	}

	cache := newMockCache()
	videoRepo := &mockVideoRepo{}
	storage := &mockObjectStorage{}
	svc := NewCourseServiceServer(courseRepo, videoRepo, cache, storage)

	req := &pb.ListCoursesRequest{Page: 1, PageSize: 10}
	concurrency := 20

	// 发起 20 个并发请求
	var wg sync.WaitGroup
	errs := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.ListCourses(context.Background(), req)
			errs <- err
		}()
	}
	wg.Wait()

	// 收集错误
	close(errs)
	successCount := 0
	for err := range errs {
		if err == nil {
			successCount++
		}
	}

	// ★ 核心断言：
	// 20 个并发请求，但因为 singleflight，
	// ListCourses 只应该被调用 1 次（或者极少数几次）
	assert.LessOrEqual(t, callCount, 3,
		"singleflight 应该将 DB 查询合并为 1~2 次，实际调用了 %d 次", callCount)
	assert.Equal(t, concurrency, successCount,
		"所有 %d 个请求都应该成功（共享同一个结果）", concurrency)
}

// ======== 新增: CreateCourse 测试 ========
func TestCourseService_CreateCourse(t *testing.T) {
	t.Run("空标题拦截", func(t *testing.T) {
		svc := NewCourseServiceServer(nil, nil, nil, nil)
		resp, err := svc.CreateCourse(context.Background(), &pb.CreateCourseRequest{Title: ""})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("数据库插入失败", func(t *testing.T) {
		repo := &mockCourseRepoForCourse{err: errors.New("db error")}
		svc := NewCourseServiceServer(repo, nil, nil, nil)
		resp, err := svc.CreateCourse(context.Background(), &pb.CreateCourseRequest{Title: "Golang"})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("成功创建", func(t *testing.T) {
		repo := &mockCourseRepoForCourse{}
		svc := NewCourseServiceServer(repo, nil, nil, nil)
		resp, err := svc.CreateCourse(context.Background(), &pb.CreateCourseRequest{Title: "Golang"})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), resp.CourseId)
	})
}

// ======== 新增: GetCoverUploadURL 测试 ========
func TestCourseService_GetCoverUploadURL(t *testing.T) {
	t.Run("空文件名拦截", func(t *testing.T) {
		svc := NewCourseServiceServer(nil, nil, nil, nil)
		resp, err := svc.GetCoverUploadURL(context.Background(), &pb.GetCoverUploadURLRequest{Filename: ""})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("生成签名URL失败", func(t *testing.T) {
		storage := &mockObjectStorage{err: errors.New("oss error")}
		svc := NewCourseServiceServer(nil, nil, nil, storage)
		resp, err := svc.GetCoverUploadURL(context.Background(), &pb.GetCoverUploadURLRequest{Filename: "cover.png"})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("成功生成签名URL", func(t *testing.T) {
		storage := &mockObjectStorage{}
		svc := NewCourseServiceServer(nil, nil, nil, storage)
		resp, err := svc.GetCoverUploadURL(context.Background(), &pb.GetCoverUploadURLRequest{Filename: "cover.png"})
		assert.NoError(t, err)
		assert.Equal(t, "https://mock-put.com", resp.UploadUrl)
	})
}

// ======== 新增: GetCourse 测试 ========
func TestCourseService_GetCourse(t *testing.T) {
	t.Run("课程不存在", func(t *testing.T) {
		repo := &mockCourseRepoForCourse{} // 空的课程库
		svc := NewCourseServiceServer(repo, nil, nil, nil)
		resp, err := svc.GetCourse(context.Background(), &pb.GetCourseRequest{CourseId: 999})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("成功获取课程和视频", func(t *testing.T) {
		repo := &mockCourseRepoForCourse{
			courses: []*model.Course{{ID: 1, Title: "Go", CoverKey: "key"}},
		}
		storage := &mockObjectStorage{}
		videoRepo := &mockVideoRepo{}
		svc := NewCourseServiceServer(repo, videoRepo, nil, storage)

		resp, err := svc.GetCourse(context.Background(), &pb.GetCourseRequest{CourseId: 1})
		assert.NoError(t, err)
		assert.NotNil(t, resp.Course)
		assert.Equal(t, "Go", resp.Course.Title)
		assert.Contains(t, resp.Course.CoverUrl, "mock-signed-url.com")
	})
}
