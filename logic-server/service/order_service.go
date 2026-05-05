package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"geekedu/common/errcode"
	pb "geekedu/common/pb"
	"geekedu/logic-server/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderServiceServer struct {
	pb.UnimplementedOrderServiceServer
	orderRepo  OrderRepository
	courseRepo CourseRepository
	cache      Cache
}

func NewOrderServiceServer(orderRepo OrderRepository, courseRepo CourseRepository, cache Cache) *OrderServiceServer {
	return &OrderServiceServer{
		orderRepo:  orderRepo,
		courseRepo: courseRepo,
		cache:      cache,
	}
}

func (s *OrderServiceServer) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	if req.UserId <= 0 || req.CourseId <= 0 {
		return nil, errcode.ErrInvalidParams.ToGRPCError()
	}

	lockKey := fmt.Sprintf("lock:order:user:%d:course:%d", req.UserId, req.CourseId)
	lockValue := uuid.New().String()
	lockAcquired := false

	if s.cache != nil {
		acquired, err := s.cache.SetNX(ctx, lockKey, lockValue, 10*time.Second)
		if err != nil {
			log.Printf("Redis SetNX error, fallback to DB unique key: %v", err)
		} else if !acquired {
			return nil, errcode.ErrOrderProcessing.ToGRPCError()
		} else {
			lockAcquired = true
		}
	}

	defer func() {
		if !lockAcquired {
			return
		}
		script := `
			if redis.call("get", KEYS[1]) == ARGV[1] then
				return redis.call("del", KEYS[1])
			else
				return 0
			end
		`
		if err := s.cache.Eval(ctx, script, []string{lockKey}, lockValue); err != nil {
			log.Printf("Redis unlock eval error: %v", err)
		}
	}()

	course, err := s.courseRepo.GetCourseByID(uint64(req.CourseId))
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Failed to get course %d: %v", req.CourseId, err)
			return nil, errcode.ErrInternal.ToGRPCError()
		}
		return nil, errcode.ErrCourseNotFound.ToGRPCError()
	}

	purchased, err := s.orderRepo.CheckPurchase(uint64(req.UserId), uint64(req.CourseId))
	if err != nil {
		log.Printf("Failed to check purchase: %v", err)
		return nil, errcode.ErrInternal.ToGRPCError()
	}
	if purchased {
		return nil, errcode.ErrAlreadyPurchased.ToGRPCError()
	}

	order := &model.Order{
		UserID:   uint64(req.UserId),
		CourseID: uint64(req.CourseId),
		Price:    course.Price,
		Status:   1,
	}

	if err := s.orderRepo.CreateOrder(order); err != nil {
		if errors.Is(err, errcode.ErrAlreadyPurchased) {
			return nil, errcode.ErrAlreadyPurchased.ToGRPCError()
		}
		log.Printf("Failed to create order: %v", err)
		return nil, errcode.ErrInternal.ToGRPCError()
	}

	return &pb.CreateOrderResponse{OrderId: int64(order.ID)}, nil
}

func (s *OrderServiceServer) CheckPurchase(ctx context.Context, req *pb.CheckPurchaseRequest) (*pb.CheckPurchaseResponse, error) {
	if req.UserId <= 0 || req.CourseId <= 0 {
		return nil, errcode.ErrInvalidParams.ToGRPCError()
	}
	purchased, err := s.orderRepo.CheckPurchase(uint64(req.UserId), uint64(req.CourseId))
	if err != nil {
		log.Printf("Failed to check purchase: %v", err)
		return nil, errcode.ErrInternal.ToGRPCError()
	}
	return &pb.CheckPurchaseResponse{Purchased: purchased}, nil
}
