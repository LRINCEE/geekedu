package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"geekedu/common/config"
	"geekedu/common/logger"
	"geekedu/common/observability"
	pb "geekedu/common/pb"
	"geekedu/logic-server/dao"
	ossutil "geekedu/logic-server/oss"
	"geekedu/logic-server/service"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	config.InitConfig("config.yaml")
	cfg := config.GetConfig()

	// Initialize the shared Zap logger before wiring dependencies.
	logger.InitLogger("dev", "logs/logic-server.log")
	defer logger.Log.Sync()

	dao.InitDB()
	cacheEnabled := os.Getenv("CACHE_ENABLED") != "false"
	if cacheEnabled {
		dao.InitRedis()
	} else {
		logger.Log.Info("Redis cache disabled by CACHE_ENABLED=false")
	}
	ossutil.InitOSSClient()

	// Wire production infrastructure dependencies.初始化生产环境基础设施依赖
	// 初始化数据库仓库
	courseRepo := dao.NewCourseDao()
	videoRepo := dao.NewVideoDao()
	orderRepo := dao.NewOrderDao()
	userRepo := dao.NewUserDao()

	// 初始化缓存
	// 初始化对象存储
	// 初始化密码管理器
	// 初始化JWT令牌提供器
	var cache service.Cache
	if cacheEnabled {
		cache = service.NewRedisAdapter(dao.RedisClient)
	}
	storage := service.NewOSSAdapter()
	pwd := service.NewBcryptPasswordManager()
	tokenProvider := service.NewJWTTokenProvider()

	// 初始化gRPC服务器
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			observability.UnaryServerTraceInterceptor(),
			observability.UnaryServerLoggingInterceptor(),
			observability.UnaryServerMetricsInterceptor(),
			observability.UnaryServerRecoveryInterceptor(),
		),
	)
	pb.RegisterUserServiceServer(grpcServer, service.NewUserServiceServer(userRepo, pwd, tokenProvider))
	pb.RegisterCourseServiceServer(grpcServer, service.NewCourseServiceServer(courseRepo, videoRepo, cache, storage))
	pb.RegisterVideoServiceServer(grpcServer, service.NewVideoServiceServer(videoRepo, orderRepo, storage))
	pb.RegisterOrderServiceServer(grpcServer, service.NewOrderServiceServer(orderRepo, courseRepo, cache))

	addr := ":" + cfg.Server.GrpcPort
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Log.Fatal("Failed to listen", zap.String("addr", addr), zap.Error(err))
	}

	metricsAddr := os.Getenv("LOGIC_METRICS_ADDR")
	if metricsAddr == "" {
		metricsAddr = ":9091"
	}
	metricsServer := &http.Server{
		Addr:    metricsAddr,
		Handler: observability.MetricsHandler(),
	}

	go func() {
		logger.Log.Info("Logic Server started", zap.String("addr", addr))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Log.Fatal("Failed to serve gRPC", zap.Error(err))
		}
	}()

	go func() {
		logger.Log.Info("Logic metrics server started", zap.String("addr", metricsAddr))
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Failed to serve metrics", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down Logic Server...")
	grpcServer.GracefulStop()
	if err := metricsServer.Shutdown(context.Background()); err != nil {
		logger.Log.Error("Failed to shutdown metrics server", zap.Error(err))
	}
	logger.Log.Info("Logic Server stopped gracefully")
}
