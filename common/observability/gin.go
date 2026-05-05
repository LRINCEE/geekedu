package observability

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(TraceHeader)
		if traceID == "" {
			traceID = GenerateTraceID()
		}

		ctx := WithTraceID(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)
		c.Set(TraceMetadataKey, traceID)
		c.Writer.Header().Set(TraceHeader, traceID)

		c.Next()
	}
}

func HTTPMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		ObserveHTTPRequest(c.Request.Method, path, c.Writer.Status(), time.Since(start))
	}
}

func PrometheusGinHandler() gin.HandlerFunc {
	handler := MetricsHandler()
	return func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}

func HealthHandler(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": service})
	}
}
