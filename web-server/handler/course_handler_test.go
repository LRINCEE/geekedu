package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"geekedu/common/pb"
	"geekedu/common/response"
	"geekedu/web-server/handler"
	"geekedu/web-server/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListCourses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 1. 使用 mockery 生成的客户端
	mockClient := mocks.NewCourseServiceClient(t)
	// 2. 预设参数与返回值
	expectedReq := &pb.ListCoursesRequest{Page: 1, PageSize: 10}
	expectedResp := &pb.ListCoursesResponse{
		Total: 100,
		Courses: []*pb.CourseItem{
			{Id: 1, Title: "Go微服务实战"},
		},
	}

	// 告诉 Mock 对象：如果收到预期请求，就返回预期响应
	mockClient.On("ListCourses", mock.Anything, expectedReq).Return(expectedResp, nil)

	// 3. 构造请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/courses?page=1&page_size=10", nil)
	c.Request = req

	// 4. 调用接口
	h := handler.NewCourseHandler(mockClient)
	h.ListCourses(c)

	// 5. 断言结果
	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response[handler.ListCoursesData]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, int64(100), resp.Data.Total)
	assert.Equal(t, "Go微服务实战", resp.Data.Courses[0].Title)

	// 验证所有 mock 方法都如期调用了
	mockClient.AssertExpectations(t)
}

func TestGetCourse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockClient := mocks.NewCourseServiceClient(t)
	expectedReq := &pb.GetCourseRequest{CourseId: 1}
	expectedResp := &pb.GetCourseResponse{
		Course: &pb.CourseItem{Id: 1, Title: "Go微服务实战"},
		Videos: []*pb.VideoItem{{Id: 101, Title: "环境搭建"}},
	}

	mockClient.On("GetCourse", mock.Anything, expectedReq).Return(expectedResp, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// 设置 Param
	c.Params = []gin.Param{{Key: "course_id", Value: "1"}}

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/courses/1", nil)
	c.Request = req

	h := handler.NewCourseHandler(mockClient)
	h.GetCourse(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response[handler.GetCourseData]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "Go微服务实战", resp.Data.Course.Title)
	assert.Equal(t, "环境搭建", resp.Data.Videos[0].Title)

	mockClient.AssertExpectations(t)
}

func TestCreateCourse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockClient := mocks.NewCourseServiceClient(t)
	reqData := handler.CreateReq{
		Title:       "新课程",
		Description: "这是一门新课程",
		Price:       99.9,
		CoverKey:    "cover.jpg",
	}

	expectedReq := &pb.CreateCourseRequest{
		Title:       "新课程",
		Description: "这是一门新课程",
		Price:       99.9,
		CoverKey:    "cover.jpg",
		TeacherId:   1001,
	}
	expectedResp := &pb.CreateCourseResponse{CourseId: 2}

	mockClient.On("CreateCourse", mock.Anything, expectedReq).Return(expectedResp, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// 模拟 JWT 解析出的 user_id
	c.Set("user_id", int64(1001))

	body, _ := json.Marshal(reqData)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/courses", bytes.NewBuffer(body))
	c.Request = req

	h := handler.NewCourseHandler(mockClient)
	h.CreateCourse(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response[handler.CreateCourseData]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, int64(2), resp.Data.CourseId)

	mockClient.AssertExpectations(t)
}
