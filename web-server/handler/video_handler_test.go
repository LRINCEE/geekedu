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

func TestInitUpload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockVideoClient := mocks.NewVideoServiceClient(t)
	reqData := handler.InitUploadRequest{
		Title:     "测试视频",
		Filename:  "test.mp4",
		PartCount: 2,
	}

	expectedReq := &pb.InitMultipartUploadRequest{
		CourseId:  1,
		Title:     "测试视频",
		Filename:  "test.mp4",
		PartCount: 2,
	}
	expectedResp := &pb.InitMultipartUploadResponse{
		UploadId:   "upload-123",
		ObjectKey:  "video/test.mp4",
		UploadUrls: []string{"url1", "url2"},
	}

	mockVideoClient.On("InitMultipartUpload", mock.Anything, expectedReq).Return(expectedResp, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = []gin.Param{{Key: "course_id", Value: "1"}}

	body, _ := json.Marshal(reqData)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/courses/1/videos/init", bytes.NewBuffer(body))
	c.Request = req

	h := handler.NewVideoHandler(mockVideoClient)
	h.InitUpload(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response[handler.InitUploadData]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "upload-123", resp.Data.UploadId)
	assert.Equal(t, 2, len(resp.Data.UploadUrls))

	mockVideoClient.AssertExpectations(t)
}

func TestGetPlayURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockVideoClient := mocks.NewVideoServiceClient(t)
	expectedReq := &pb.GetVideoPlayURLRequest{
		VideoId: 10,
		UserId:  1001,
	}
	expectedResp := &pb.GetVideoPlayURLResponse{
		PlayUrl: "http://oss.example.com/play/10",
	}

	mockVideoClient.On("GetVideoPlayURL", mock.Anything, expectedReq).Return(expectedResp, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = []gin.Param{{Key: "video_id", Value: "10"}}
	c.Set("user_id", int64(1001))

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/videos/10/play_url", nil)
	c.Request = req

	h := handler.NewVideoHandler(mockVideoClient)
	h.GetPlayURL(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response[handler.GetPlayURLData]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "http://oss.example.com/play/10", resp.Data.PlayUrl)

	mockVideoClient.AssertExpectations(t)
}
