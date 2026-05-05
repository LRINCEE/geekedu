package middleware

import (
	"strings"

	"geekedu/common/errcode"
	"geekedu/common/jwt"
	"geekedu/common/response"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头中获取Authorization字段
		authHeader := c.GetHeader("Authorization")
		// 检查Authorization字段是否存在
		if authHeader == "" {
			response.Error(c, errcode.ErrUnauthorized)
			c.Abort()
			return
		}

		// 验证Authorization字段格式是否为"Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		// 检查Authorization字段格式是否正确，即是否包含"Bearer"和token
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Error(c, errcode.ErrUnauthorized)
			c.Abort()
			return
		}
		// 验证JWT令牌
		claims, err := jwt.ParseToken(parts[1])
		if err != nil {
			response.Error(c, errcode.ErrUnauthorized)
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// AdminMiddleware 检查用户是否为管理员角色
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role.(int32) != 1 {
			response.Error(c, errcode.ErrForbidden)
			c.Abort()
			return
		}
		c.Next()
	}
}
