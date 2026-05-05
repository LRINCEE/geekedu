package service

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"

	"geekedu/common/errcode"
	pb "geekedu/common/pb"
	"geekedu/logic-server/model"

	"gorm.io/gorm"
)

type VideoServiceServer struct {
	pb.UnimplementedVideoServiceServer
	videoRepo VideoRepository
	orderRepo OrderRepository
	storage   ObjectStorage
}

func NewVideoServiceServer(videoRepo VideoRepository, orderRepo OrderRepository, storage ObjectStorage) *VideoServiceServer {
	return &VideoServiceServer{
		videoRepo: videoRepo,
		orderRepo: orderRepo,
		storage:   storage,
	}
}

func (s *VideoServiceServer) InitMultipartUpload(ctx context.Context, req *pb.InitMultipartUploadRequest) (*pb.InitMultipartUploadResponse, error) {
	if req.CourseId <= 0 || req.Title == "" || req.Filename == "" || req.PartCount <= 0 || req.PartCount > 1000 {
		return nil, errcode.ErrInvalidParams.ToGRPCError()
	}

	objectKey := s.storage.GenerateVideoKey(uint64(req.CourseId), req.Filename)
	uploadID, err := s.storage.InitiateMultipartUpload(objectKey)
	if err != nil {
		log.Printf("Failed to initiate multipart upload: %v", err)
		return nil, errcode.ErrOSS.ToGRPCError()
	}

	uploadURLs := make([]string, 0, int(req.PartCount))
	for i := 1; i <= int(req.PartCount); i++ {
		url, err := s.storage.GeneratePresignedPartURL(objectKey, uploadID, i)
		if err != nil {
			log.Printf("Failed to generate presigned URL for part %d: %v", i, err)
			return nil, errcode.ErrOSS.ToGRPCError()
		}
		uploadURLs = append(uploadURLs, url)
	}

	return &pb.InitMultipartUploadResponse{
		UploadId:   uploadID,
		ObjectKey:  objectKey,
		UploadUrls: uploadURLs,
	}, nil
}

func (s *VideoServiceServer) CompleteMultipartUpload(ctx context.Context, req *pb.CompleteMultipartUploadRequest) (*pb.CompleteMultipartUploadResponse, error) {
	if req.CourseId <= 0 || req.UploadId == "" || req.ObjectKey == "" || req.Title == "" || len(req.Parts) == 0 {
		return nil, errcode.ErrInvalidParams.ToGRPCError()
	}
	if !strings.HasPrefix(req.ObjectKey, videoObjectPrefix(uint64(req.CourseId))) {
		return nil, errcode.ErrInvalidParams.ToGRPCError()
	}

	parts := make([]UploadPart, len(req.Parts))
	for i, p := range req.Parts {
		if p.PartNumber <= 0 || p.Etag == "" {
			return nil, errcode.ErrInvalidParams.ToGRPCError()
		}
		parts[i] = UploadPart{PartNumber: int(p.PartNumber), ETag: p.Etag}
	}

	if err := s.storage.CompleteMultipartUpload(req.ObjectKey, req.UploadId, parts); err != nil {
		log.Printf("Failed to complete multipart upload: %v", err)
		return nil, errcode.ErrOSS.ToGRPCError()
	}

	video := &model.CourseVideo{
		CourseID: uint64(req.CourseId),
		Title:    req.Title,
		VideoKey: req.ObjectKey,
	}
	if err := s.videoRepo.CreateVideo(video); err != nil {
		log.Printf("Failed to save video record: %v", err)
		return nil, errcode.ErrInternal.ToGRPCError()
	}

	return &pb.CompleteMultipartUploadResponse{VideoId: int64(video.ID)}, nil
}

func (s *VideoServiceServer) GetVideoPlayURL(ctx context.Context, req *pb.GetVideoPlayURLRequest) (*pb.GetVideoPlayURLResponse, error) {
	if req.VideoId <= 0 || req.UserId <= 0 {
		return nil, errcode.ErrInvalidParams.ToGRPCError()
	}

	video, err := s.videoRepo.GetVideoByID(uint64(req.VideoId))
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Failed to get video %d: %v", req.VideoId, err)
			return nil, errcode.ErrInternal.ToGRPCError()
		}
		return nil, errcode.ErrVideoNotFound.ToGRPCError()
	}

	purchased, err := s.orderRepo.CheckPurchase(uint64(req.UserId), video.CourseID)
	if err != nil {
		log.Printf("Failed to check purchase: %v", err)
		return nil, errcode.ErrInternal.ToGRPCError()
	}
	if !purchased {
		return nil, errcode.ErrNotPurchased.ToGRPCError()
	}

	playURL, err := s.storage.GenerateSignedURL(video.VideoKey, 3600)
	if err != nil {
		log.Printf("Failed to generate play URL: %v", err)
		return nil, errcode.ErrInternal.ToGRPCError()
	}

	return &pb.GetVideoPlayURLResponse{PlayUrl: playURL}, nil
}

func videoObjectPrefix(courseID uint64) string {
	return "videos/" + strconv.FormatUint(courseID, 10) + "/"
}
