package handler

import (
	"context"
	"time"

	"geekedu/common/errcode"
	pb "geekedu/common/pb"
	"geekedu/common/response"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	orderClient pb.OrderServiceClient
}

func NewOrderHandler(orderClient pb.OrderServiceClient) *OrderHandler {
	return &OrderHandler{orderClient: orderClient}
}

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	CourseID int64 `json:"course_id" binding:"required" example:"1001"` // 必须传课程ID
}

// CreateOrderData 创建订单返回数据
type CreateOrderData struct {
	OrderId int64 `json:"order_id"`
}

// CreateOrder 创建订单
// @Summary 创建订单
// @Description 用户购买一门课程
// @Tags 订单模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body CreateOrderRequest true "购买请求"
// @Success 200 {object} response.Response[CreateOrderData] "成功"
// @Failure 400 {object} response.Response[any] "参数错误或已购买"
// @Failure 500 {object} response.Response[any] "内部错误"
// @Router /api/v1/orders [post]
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	// 2. 绑定前端传来的参数到req对象
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrInvalidParams)
		return
	}

	userID, _ := c.Get("user_id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.orderClient.CreateOrder(ctx, &pb.CreateOrderRequest{
		UserId:   userID.(int64),
		CourseId: req.CourseID,
	})
	if err != nil {
		handleGRPCError(c, err)
		return
	}

	response.Success(c, CreateOrderData{OrderId: resp.OrderId})
}
