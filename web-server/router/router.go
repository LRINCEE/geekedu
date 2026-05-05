package router

import (
	_ "geekedu/web-server/docs"
	"geekedu/web-server/grpc_client"
	"geekedu/web-server/handler"
	"geekedu/web-server/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/time/rate"
)

func SetupRouter() *gin.Engine {
	r := gin.New()
	r.Use(middleware.GinLogger(), middleware.GinRecovery(true))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "geekedu-web"})
	})

	r.Use(middleware.CORSMiddleware())
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	authHandler := handler.NewAuthHandler(grpc_client.UserClient)
	courseHandler := handler.NewCourseHandler(grpc_client.CourseClient)
	videoHandler := handler.NewVideoHandler(grpc_client.VideoClient)
	orderHandler := handler.NewOrderHandler(grpc_client.OrderClient)

	v1 := r.Group("/api/v1")
	{
		authLimiter := middleware.RateLimiter(rate.Limit(10000), 10000)
		writeLimiter := middleware.RateLimiter(rate.Limit(10000), 10000)

		v1.POST("/auth/register", authLimiter, authHandler.Register)
		v1.POST("/auth/login", authLimiter, authHandler.Login)
		v1.GET("/courses", courseHandler.ListCourses)
		v1.GET("/courses/:course_id", courseHandler.GetCourse)

		auth := v1.Group("")
		auth.Use(middleware.AuthMiddleware())
		{
			auth.POST("/orders", writeLimiter, orderHandler.CreateOrder)
			auth.GET("/player/:video_id", videoHandler.GetPlayURL)
		}

		admin := v1.Group("")
		admin.Use(writeLimiter, middleware.AuthMiddleware(), middleware.AdminMiddleware())
		{
			admin.POST("/courses", courseHandler.CreateCourse)
			admin.GET("/courses/cover/upload_url", courseHandler.GetCoverUploadURL)
			admin.POST("/courses/:course_id/videos/init", videoHandler.InitUpload)
			admin.POST("/courses/:course_id/videos/complete", videoHandler.CompleteUpload)
		}
	}

	return r
}
