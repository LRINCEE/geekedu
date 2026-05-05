package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"geekedu/common/errcode"
	pb "geekedu/common/pb"
	"geekedu/common/response"

	"github.com/gin-gonic/gin"
)

type CourseHandler struct {
	courseClient pb.CourseServiceClient
}

func NewCourseHandler(courseClient pb.CourseServiceClient) *CourseHandler {
	return &CourseHandler{courseClient: courseClient}
}

type ListCoursesData struct {
	Courses []*pb.CourseItem `json:"courses"`
	Total   int64            `json:"total"`
}

// ListCourses 获取课程列表
func (h *CourseHandler) ListCourses(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.courseClient.ListCourses(ctx, &pb.ListCoursesRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, ListCoursesData{Courses: resp.Courses, Total: resp.Total})
}

type GetCoverUploadURLData struct {
	UploadUrl string `json:"upload_url"`
	CoverKey  string `json:"cover_key"`
}

// GetCoverUploadURL 获取课程上传封面的URL
func (h *CourseHandler) GetCoverUploadURL(c *gin.Context) {
	// 从请求参数中获取文件名
	filename := c.Query("filename")
	if filename == "" {
		response.Error(c, errcode.ErrInvalidParams)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.courseClient.GetCoverUploadURL(ctx, &pb.GetCoverUploadURLRequest{Filename: filename})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, GetCoverUploadURLData{UploadUrl: resp.UploadUrl, CoverKey: resp.CoverKey})
}

type CreateReq struct {
	Title       string  `json:"title" binding:"required" example:"Go course"`
	Description string  `json:"description" example:"Introductory Go course"`
	Price       float64 `json:"price" example:"99.9"`
	CoverKey    string  `json:"cover_key" example:"covers/xxx.jpg"`
}

type CreateCourseData struct {
	CourseId int64 `json:"course_id"`
}

// CreateCourse 创建课程
func (h *CourseHandler) CreateCourse(c *gin.Context) {
	req, err := h.bindCreateCourseRequest(c)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParams)
		return
	}

	teacherID, _ := c.Get("user_id")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.courseClient.CreateCourse(ctx, &pb.CreateCourseRequest{
		Title:       req.Title,
		Description: req.Description,
		Price:       req.Price,
		CoverKey:    req.CoverKey,
		TeacherId:   teacherID.(int64),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, CreateCourseData{CourseId: resp.CourseId})
}

// bindCreateCourseRequest 绑定创建课程的请求参数(不带封面)
func (h *CourseHandler) bindCreateCourseRequest(c *gin.Context) (CreateReq, error) {
	// 检查请求体是否为multipart/form-data
	contentType := c.GetHeader("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		return h.bindCreateCourseMultipart(c)
	}

	var req CreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return CreateReq{}, err
	}
	if req.Title == "" || req.Price < 0 {
		return CreateReq{}, fmt.Errorf("invalid course payload")
	}
	return req, nil
}

// bindCreateCourseMultipart 绑定创建课程的multipart/form-data请求参数(带封面)
func (h *CourseHandler) bindCreateCourseMultipart(c *gin.Context) (CreateReq, error) {
	price, err := strconv.ParseFloat(c.PostForm("price"), 64)
	if err != nil {
		return CreateReq{}, err
	}

	req := CreateReq{
		Title:       c.PostForm("title"),
		Description: c.PostForm("description"),
		Price:       price,
		CoverKey:    c.PostForm("cover_key"),
	}
	if req.Title == "" || req.Price < 0 {
		return CreateReq{}, fmt.Errorf("invalid course form")
	}

	fileHeader, err := c.FormFile("cover")
	if err == nil && fileHeader != nil && req.CoverKey == "" {
		coverKey, err := h.uploadCoverViaSignedURL(fileHeader.Filename, func() (io.ReadCloser, error) {
			return fileHeader.Open()
		})
		if err != nil {
			return CreateReq{}, err
		}
		req.CoverKey = coverKey
	}
	return req, nil
}

// uploadCoverViaSignedURL 上传课程封面到指定的URL
func (h *CourseHandler) uploadCoverViaSignedURL(filename string, openFile func() (io.ReadCloser, error)) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uploadResp, err := h.courseClient.GetCoverUploadURL(ctx, &pb.GetCoverUploadURLRequest{Filename: filename})
	if err != nil {
		return "", err
	}

	file, err := openFile()
	if err != nil {
		return "", err
	}
	defer file.Close()

	putReq, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadResp.UploadUrl, file)
	if err != nil {
		return "", err
	}
	putReq.Header.Set("Content-Type", "")

	httpClient := &http.Client{Timeout: 10 * time.Second}
	putResp, err := httpClient.Do(putReq)
	if err != nil {
		return "", err
	}
	defer putResp.Body.Close()
	if putResp.StatusCode < http.StatusOK || putResp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("upload cover failed: status=%d", putResp.StatusCode)
	}
	return uploadResp.CoverKey, nil
}

type GetCourseData struct {
	Course *pb.CourseItem  `json:"course"`
	Videos []*pb.VideoItem `json:"videos"`
}

func (h *CourseHandler) GetCourse(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("course_id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParams)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.courseClient.GetCourse(ctx, &pb.GetCourseRequest{CourseId: courseID})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, GetCourseData{Course: resp.Course, Videos: resp.Videos})
}
