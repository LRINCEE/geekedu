package middleware

import (
	"net/http"

	"geekedu/common/logger"
	"geekedu/common/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// RateLimiter 基于 x/time/rate (令牌桶算法) 的全局限流器
// rateLimit: 每秒产生的令牌数 (QPS)
// capacity: 桶的容量 (允许的最大突发并发量)
func RateLimiter(rateLimit rate.Limit, capacity int) gin.HandlerFunc {
	// 创建一个全局的令牌桶
	// 注意：在真实的分布式集群中，这里应该换成基于 Redis 的分布式限流（如 Redis Cell 或 Lua 脚本）
	// 目前由于 web-server 是单节点，使用单机内存限流器即可
	limiter := rate.NewLimiter(rateLimit, capacity)

	return func(c *gin.Context) {
		// Allow() 尝试从桶中消耗一个令牌
		if !limiter.Allow() {
			logger.Log.Warn("Rate limit exceeded",
				zap.String("ip", c.ClientIP()),
				zap.String("path", c.Request.URL.Path),
			)
			// 返回 429 Too Many Requests
			c.JSON(http.StatusTooManyRequests, response.Response[any]{
				Code: http.StatusTooManyRequests,
				Msg:  "服务器繁忙，请稍后再试",
				Data: nil,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
