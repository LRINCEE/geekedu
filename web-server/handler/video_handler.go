package handler

import (
	"context"
	"strconv"
	"time"

	"geekedu/common/errcode"
	pb "geekedu/common/pb"
	"geekedu/common/response"

	"github.com/gin-gonic/gin"
)

type VideoHandler struct {
	videoClient pb.VideoServiceClient
}

func NewVideoHandler(videoClient pb.VideoServiceClient) *VideoHandler {
	return &VideoHandler{videoClient: videoClient}
}

// InitUploadRequest 初始化分片上传请求
type InitUploadRequest struct {
	Title     string `json:"title" binding:"required" example:"第一节：Go语言基础"`
	Filename  string `json:"filename" binding:"required" example:"lesson1.mp4"`
	PartCount int32  `json:"part_count" binding:"required" example:"3"`
}

// CompleteUploadRequest 完成分片上传请求
type CompleteUploadRequest struct {
	UploadID  string     `json:"upload_id" binding:"required"`
	ObjectKey string     `json:"object_key" binding:"required"`
	Title     string     `json:"title" binding:"required" example:"第一节：Go语言基础"`
	Parts     []PartInfo `json:"parts" binding:"required"`
}

// PartInfo 分片信息
type PartInfo struct {
	PartNumber int32  `json:"part_number" example:"1"`
	ETag       string `json:"etag" example:"\"d41d8cd98f00b204e9800998ecf8427e\""`
}

// InitUploadData 初始化上传返回数据
type InitUploadData struct {
	UploadId   string   `json:"upload_id"`
	ObjectKey  string   `json:"object_key"`
	UploadUrls []string `json:"upload_urls"`
}

// InitUpload 初始化视频分片上传
// @Summary 初始化视频分片上传
// @Description 教师上传视频前获取OSS分片预签名URL列表
// @Tags 视频模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param course_id path int true "课程 ID"
// @Param request body InitUploadRequest true "初始化上传信息"
// @Success 200 {object} response.Response[InitUploadData] "成功"
// @Failure 400 {object} response.Response[any] "参数错误"
// @Failure 500 {object} response.Response[any] "内部错误"
// @Router /api/v1/courses/{course_id}/videos/init [post]
func (h *VideoHandler) InitUpload(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("course_id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParams)
		return
	}

	var req InitUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrInvalidParams)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	resp, err := h.videoClient.InitMultipartUpload(ctx, &pb.InitMultipartUploadRequest{
		CourseId:  courseID,
		Title:     req.Title,
		Filename:  req.Filename,
		PartCount: req.PartCount,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, InitUploadData{
		UploadId:   resp.UploadId,
		ObjectKey:  resp.ObjectKey,
		UploadUrls: resp.UploadUrls,
	})
}

// CompleteUploadData 完成上传返回数据
type CompleteUploadData struct {
	VideoId int64 `json:"video_id"`
}

// CompleteUpload 完成视频分片上传
// @Summary 完成视频分片上传
// @Description 教师完成OSS视频分片上传后通知后端合并
// @Tags 视频模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param course_id path int true "课程 ID"
// @Param request body CompleteUploadRequest true "分片合并信息"
// @Success 200 {object} response.Response[CompleteUploadData] "成功"
// @Failure 400 {object} response.Response[any] "参数错误"
// @Failure 500 {object} response.Response[any] "内部错误"
// @Router /api/v1/courses/{course_id}/videos/complete [post]
func (h *VideoHandler) CompleteUpload(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("course_id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParams)
		return
	}

	var req CompleteUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrInvalidParams)
		return
	}

	parts := make([]*pb.PartInfo, len(req.Parts))
	for i, p := range req.Parts {
		parts[i] = &pb.PartInfo{
			PartNumber: p.PartNumber,
			Etag:       p.ETag,
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	resp, err := h.videoClient.CompleteMultipartUpload(ctx, &pb.CompleteMultipartUploadRequest{
		CourseId:  courseID,
		UploadId:  req.UploadID,
		ObjectKey: req.ObjectKey,
		Title:     req.Title,
		Parts:     parts,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, CompleteUploadData{VideoId: resp.VideoId})
}

// GetPlayURLData 获取播放链接返回数据
type GetPlayURLData struct {
	PlayUrl string `json:"play_url"`
}

// GetPlayURL 获取视频播放链接
// @Summary 获取视频播放链接
// @Description 已购买课程的用户获取视频预签名播放链接
// @Tags 视频模块
// @Produce json
// @Security Bearer
// @Param video_id path int true "视频 ID"
// @Success 200 {object} response.Response[GetPlayURLData] "成功"
// @Failure 400 {object} response.Response[any] "参数错误"
// @Failure 403 {object} response.Response[any] "未购买课程"
// @Failure 500 {object} response.Response[any] "内部错误"
// @Router /api/v1/videos/{video_id}/play_url [get]
func (h *VideoHandler) GetPlayURL(c *gin.Context) {
	videoID, err := strconv.ParseInt(c.Param("video_id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParams)
		return
	}

	userID, _ := c.Get("user_id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	resp, err := h.videoClient.GetVideoPlayURL(ctx, &pb.GetVideoPlayURLRequest{
		VideoId: videoID,
		UserId:  userID.(int64),
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, GetPlayURLData{PlayUrl: resp.PlayUrl})
}
