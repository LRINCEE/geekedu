package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"geekedu/common/config"
	"geekedu/common/logger"
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
	grpcServer := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcServer, service.NewUserServiceServer(userRepo, pwd, tokenProvider))
	pb.RegisterCourseServiceServer(grpcServer, service.NewCourseServiceServer(courseRepo, videoRepo, cache, storage))
	pb.RegisterVideoServiceServer(grpcServer, service.NewVideoServiceServer(videoRepo, orderRepo, storage))
	pb.RegisterOrderServiceServer(grpcServer, service.NewOrderServiceServer(orderRepo, courseRepo, cache))

	addr := ":" + cfg.Server.GrpcPort
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Log.Fatal("Failed to listen", zap.String("addr", addr), zap.Error(err))
	}

	go func() {
		logger.Log.Info("Logic Server started", zap.String("addr", addr))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Log.Fatal("Failed to serve gRPC", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down Logic Server...")
	grpcServer.GracefulStop()
	logger.Log.Info("Logic Server stopped gracefully")
}
