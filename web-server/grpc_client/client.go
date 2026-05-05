package grpc_client

import (
	"log"

	"geekedu/common/config"
	pb "geekedu/common/pb"

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
	)
	if err != nil {
		log.Fatalf("Failed to connect to Logic Server at %s: %v", addr, err)
	}

	UserClient = pb.NewUserServiceClient(conn)
	CourseClient = pb.NewCourseServiceClient(conn)
	VideoClient = pb.NewVideoServiceClient(conn)
	OrderClient = pb.NewOrderServiceClient(conn)

	log.Printf("gRPC client connected to Logic Server at %s", addr)
}

func Close() {
	if conn != nil {
		conn.Close()
	}
}
