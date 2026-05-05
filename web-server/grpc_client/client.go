package grpc_client

import (
	"geekedu/common/config"
	"geekedu/common/logger"
	"geekedu/common/observability"
	pb "geekedu/common/pb"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	conn         *grpc.ClientConn
	UserClient   pb.UserServiceClient
	CourseClient pb.CourseServiceClient
	VideoClient  pb.VideoServiceClient
	OrderClient  pb.OrderServiceClient
)

func InitGRPCClient() {
	cfg := config.GetConfig()
	addr := cfg.LogicServer.Addr

	var err error
	conn, err = grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			observability.UnaryClientTraceInterceptor(),
			observability.UnaryClientLoggingInterceptor(),
			observability.UnaryClientMetricsInterceptor(),
		),
	)
	if err != nil {
		logger.Log.Fatal("Failed to connect to Logic Server",
			zap.String("addr", addr),
			zap.Error(err),
		)
	}

	UserClient = pb.NewUserServiceClient(conn)
	CourseClient = pb.NewCourseServiceClient(conn)
	VideoClient = pb.NewVideoServiceClient(conn)
	OrderClient = pb.NewOrderServiceClient(conn)

	logger.Log.Info("gRPC client connected to Logic Server", zap.String("addr", addr))
}

func Close() {
	if conn != nil {
		conn.Close()
	}
}
