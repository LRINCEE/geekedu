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

func TestCreateOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockOrderClient := mocks.NewOrderServiceClient(t)
	reqData := handler.CreateOrderRequest{
		CourseID: 101,
	}

	expectedReq := &pb.CreateOrderRequest{
		UserId:   1001,
		CourseId: 101,
	}
	expectedResp := &pb.CreateOrderResponse{
		OrderId: 2001,
	}

	mockOrderClient.On("CreateOrder", mock.Anything, expectedReq).Return(expectedResp, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// 模拟 JWT 解析出的 user_id
	c.Set("user_id", int64(1001))

	body, _ := json.Marshal(reqData)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBuffer(body))
	c.Request = req

	h := handler.NewOrderHandler(mockOrderClient)
	h.CreateOrder(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response[handler.CreateOrderData]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, int64(2001), resp.Data.OrderId)

	mockOrderClient.AssertExpectations(t)
}
