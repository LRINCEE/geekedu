package oss

import (
	"bytes"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"geekedu/common/config"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var (
	client *oss.Client
	bucket *oss.Bucket
)

func InitOSSClient() {
	cfg := config.GetConfig()

	var err error
	client, err = oss.New(cfg.OSS.Endpoint, cfg.OSS.AccessKeyID, cfg.OSS.AccessKeySecret)
	if err != nil {
		log.Fatalf("Failed to create OSS client: %v", err)
	}

	bucket, err = client.Bucket(cfg.OSS.BucketName)
	if err != nil {
		log.Fatalf("Failed to get OSS bucket: %v", err)
	}

	log.Printf("OSS client initialized, bucket: %s", cfg.OSS.BucketName)
}

// UploadFile uploads data to OSS with the given object key
func UploadFile(objectKey string, data []byte) error {
	return bucket.PutObject(objectKey, bytes.NewReader(data))
}

// GenerateSignedURL generates a presigned GET URL for the object
func GenerateSignedURL(objectKey string, expireSec int64) (string, error) {
	url, err := bucket.SignURL(objectKey, oss.HTTPGet, expireSec)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}
	return url, nil
}

// GeneratePresignedPutURL generates a presigned PUT URL for single object upload
func GeneratePresignedPutURL(objectKey string, expireSec int64) (string, error) {
	url, err := bucket.SignURL(objectKey, oss.HTTPPut, expireSec)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned PUT URL: %w", err)
	}
	return url, nil
}

// InitiateMultipartUpload starts a multipart upload and returns the upload ID
func InitiateMultipartUpload(objectKey string) (string, error) {
	result, err := bucket.InitiateMultipartUpload(objectKey)
	if err != nil {
		return "", fmt.Errorf("failed to initiate multipart upload: %w", err)
	}
	return result.UploadID, nil
}

// GeneratePresignedPartURL generates a presigned PUT URL for uploading a part
func GeneratePresignedPartURL(objectKey, uploadID string, partNumber int) (string, error) {
	options := []oss.Option{
		oss.AddParam("uploadId", uploadID),
		oss.AddParam("partNumber", fmt.Sprintf("%d", partNumber)),
	}
	url, err := bucket.SignURL(objectKey, oss.HTTPPut, 3600, options...)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned part URL: %w", err)
	}
	return url, nil
}

// CompleteMultipartUpload finishes the multipart upload
func CompleteMultipartUpload(objectKey, uploadID string, parts []UploadPart) error {
	imur := oss.InitiateMultipartUploadResult{
		Bucket:   bucket.BucketName,
		Key:      objectKey,
		UploadID: uploadID,
	}

	ossParts := make([]oss.UploadPart, len(parts))
	for i, p := range parts {
		ossParts[i] = oss.UploadPart{
			PartNumber: p.PartNumber,
			ETag:       p.ETag,
		}
	}

	_, err := bucket.CompleteMultipartUpload(imur, ossParts)
	if err != nil {
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}
	return nil
}

// GenerateCoverKey generates an OSS object key for a course cover image
func GenerateCoverKey(courseID uint64, ext string) string {
	return fmt.Sprintf("covers/%d_%d%s", courseID, time.Now().UnixMilli(), ext)
}

// GenerateVideoKey generates an OSS object key for a course video
func GenerateVideoKey(courseID uint64, filename string) string {
	safeName := strings.ReplaceAll(filepath.Base(filename), " ", "_")
	return fmt.Sprintf("videos/%d/%d_%s", courseID, time.Now().UnixMilli(), safeName)
}

type UploadPart struct {
	PartNumber int
	ETag       string
}
