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

func TestAuthRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 1. 初始化 Mock 客户端
	mockUserClient := mocks.NewUserServiceClient(t)

	// 2. 配置 Mock 行为
	reqData := handler.RegisterRequest{
		Username: "testuser",
		Password: "password123",
	}
	expectedReq := &pb.RegisterRequest{
		Username: "testuser",
		Password: "password123",
	}
	expectedResp := &pb.RegisterResponse{
		UserId: 1001,
	}

	// 拦截 Register 调用并返回成功响应
	mockUserClient.On("Register", mock.Anything, expectedReq).Return(expectedResp, nil)

	// 3. 构造 HTTP 请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body, _ := json.Marshal(reqData)
	req, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
	c.Request = req

	// 4. 调用目标 Handler
	h := handler.NewAuthHandler(mockUserClient)
	h.Register(c)

	// 5. 断言验证
	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response[handler.RegisterData]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, int64(1001), resp.Data.UserId)

	mockUserClient.AssertExpectations(t)
}

func TestAuthLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockUserClient := mocks.NewUserServiceClient(t)

	reqData := handler.LoginRequest{
		Username: "testuser",
		Password: "password123",
	}
	expectedReq := &pb.LoginRequest{
		Username: "testuser",
		Password: "password123",
	}
	expectedResp := &pb.LoginResponse{
		Token:  "mock-jwt-token",
		UserId: 1001,
		Role:   1,
	}

	mockUserClient.On("Login", mock.Anything, expectedReq).Return(expectedResp, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body, _ := json.Marshal(reqData)
	req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
	c.Request = req

	h := handler.NewAuthHandler(mockUserClient)
	h.Login(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response[handler.LoginData]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "mock-jwt-token", resp.Data.Token)
	assert.Equal(t, int64(1001), resp.Data.UserId)
	assert.Equal(t, int32(1), resp.Data.Role)

	mockUserClient.AssertExpectations(t)
}
