package handler

import (
	"context"
	"time"

	"geekedu/common/errcode"
	pb "geekedu/common/pb"
	"geekedu/common/response"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	userClient pb.UserServiceClient // 从字段注入，而不是读全局变量
}

func NewAuthHandler(userClient pb.UserServiceClient) *AuthHandler {
	return &AuthHandler{userClient: userClient}
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"admin"`
	Password string `json:"password" binding:"required" example:"123456"`
}

// Swagger 专用的 Data 结构体 (由于匿名结构体无法被 swag 解析)
type RegisterData struct {
	UserId int64 `json:"user_id" example:"1001"`
}

type LoginData struct {
	Token  string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	UserId int64  `json:"user_id" example:"1001"`
	Role   int32  `json:"role" example:"1"`
}

// Register godoc
// @Summary 用户注册
// @Description 提交用户名和密码进行注册
// @Tags 认证中心
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "注册参数"
// @Success 200 {object} response.Response[RegisterData] "注册成功"
// @Failure 400 {object} response.Response[any] "参数错误"
// @Failure 409 {object} response.Response[any] "用户名已存在"
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrInvalidParams)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	resp, err := h.userClient.Register(ctx, &pb.RegisterRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, RegisterData{UserId: resp.UserId})
}

// Login godoc
// @Summary 用户登录
// @Description 提交用户名和密码获取 JWT Token
// @Tags 认证中心
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录参数"
// @Success 200 {object} response.Response[LoginData] "登录成功"
// @Failure 400 {object} response.Response[any] "参数错误"
// @Failure 401 {object} response.Response[any] "用户名或密码错误"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrInvalidParams)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	resp, err := h.userClient.Login(ctx, &pb.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, LoginData{
		Token:  resp.Token,
		UserId: resp.UserId,
		Role:   resp.Role,
	})
}

func handleGRPCError(c *gin.Context, err error) {
	bizErr := errcode.FromGRPCError(err)
	response.Error(c, bizErr)
}
