package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"geekedu/common/config"
	"geekedu/common/logger"
	"geekedu/web-server/grpc_client"
	"geekedu/web-server/router"

	"go.uber.org/zap"
)

// @title GeekEdu API 接口文档
// @version 1.0
// @description GeekEdu 微服务在线教育平台 API 网关文档
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	config.InitConfig("config.yaml")
	cfg := config.GetConfig()

	// 初始化 Zap 全局日志
	logger.InitLogger("dev", "logs/web-server.log")
	defer logger.Log.Sync()

	grpc_client.InitGRPCClient()
	defer grpc_client.Close()

	r := router.SetupRouter()

	addr := ":" + cfg.Server.HttpPort
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		logger.Log.Info("Web Server started", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down Web Server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Error("Server forced to shutdown", zap.Error(err))
	}
}
