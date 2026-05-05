package response

import (
	"geekedu/common/errcode"

	"github.com/gin-gonic/gin"
)

// Response 泛型结构体，解决 Swagger 文档无法推断 interface{} 类型的痛点
type Response[T any] struct {
	Code int    `json:"code" example:"0"`
	Msg  string `json:"msg" example:"success"`
	Data T      `json:"data"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(errcode.Success.HttpStatus, Response[interface{}]{
		Code: errcode.Success.Code,
		Msg:  errcode.Success.Msg,
		Data: data,
	})
}

func Error(c *gin.Context, err *errcode.ErrCode) {
	c.JSON(err.HttpStatus, Response[interface{}]{
		Code: err.Code,
		Msg:  err.Msg,
		Data: nil,
	})
}

func ErrorWithMsg(c *gin.Context, err *errcode.ErrCode, msg string) {
	c.JSON(err.HttpStatus, Response[interface{}]{
		Code: err.Code,
		Msg:  msg,
		Data: nil,
	})
}
