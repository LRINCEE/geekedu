package service

import (
	"context"
	"errors"
	"log"

	"geekedu/common/errcode"
	pb "geekedu/common/pb"
	"geekedu/logic-server/model"

	"gorm.io/gorm"
)

type UserServiceServer struct {
	pb.UnimplementedUserServiceServer
	userRepo UserRepository
	pwd      PasswordManager
	token    TokenProvider
}

func NewUserServiceServer(userRepo UserRepository, pwd PasswordManager, token TokenProvider) *UserServiceServer {
	return &UserServiceServer{
		userRepo: userRepo,
		pwd:      pwd,
		token:    token,
	}
}

func (s *UserServiceServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req.Username == "" || req.Password == "" {
		return nil, errcode.ErrInvalidParams.ToGRPCError()
	}

	existing, err := s.userRepo.GetUserByUsername(req.Username)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Failed to check existing user: %v", err)
		return nil, errcode.ErrInternal.ToGRPCError()
	}
	if existing != nil {
		return nil, errcode.ErrUsernameExists.ToGRPCError()
	}

	hashedPassword, err := s.pwd.Hash(req.Password)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		return nil, errcode.ErrInternal.ToGRPCError()
	}

	user := &model.User{
		Username: req.Username,
		Password: hashedPassword,
		Role:     0,
	}

	if err := s.userRepo.CreateUser(user); err != nil {
		if errors.Is(err, errcode.ErrUsernameExists) {
			return nil, errcode.ErrUsernameExists.ToGRPCError()
		}
		log.Printf("Failed to create user: %v", err)
		return nil, errcode.ErrInternal.ToGRPCError()
	}

	return &pb.RegisterResponse{UserId: int64(user.ID)}, nil
}

func (s *UserServiceServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.Username == "" || req.Password == "" {
		return nil, errcode.ErrInvalidParams.ToGRPCError()
	}

	user, err := s.userRepo.GetUserByUsername(req.Username)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Failed to get user: %v", err)
			return nil, errcode.ErrInternal.ToGRPCError()
		}
		return nil, errcode.ErrInvalidCredential.ToGRPCError()
	}

	if err := s.pwd.Compare(user.Password, req.Password); err != nil {
		return nil, errcode.ErrInvalidCredential.ToGRPCError()
	}

	token, err := s.token.Generate(int64(user.ID), int32(user.Role))
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		return nil, errcode.ErrInternal.ToGRPCError()
	}

	return &pb.LoginResponse{
		Token:  token,
		UserId: int64(user.ID),
		Role:   int32(user.Role),
	}, nil
}
