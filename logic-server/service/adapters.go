package service

import (
	"context"
	"errors"
	"time"

	"geekedu/common/jwt"
	ossutil "geekedu/logic-server/oss"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// RedisAdapter 将 redis.Client 适配为 Cache 接口
type RedisAdapter struct {
	client *redis.Client
}

func NewRedisAdapter(client *redis.Client) *RedisAdapter {
	return &RedisAdapter{client: client}
}

func (r *RedisAdapter) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if r.client == nil {
		return nil, false, errors.New("redis client is nil")
	}
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return []byte(val), true, nil
}

func (r *RedisAdapter) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	if r.client == nil {
		return errors.New("redis client is nil")
	}
	return r.client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisAdapter) SetNX(ctx context.Context, key string, value string, expiration time.Duration) (bool, error) {
	if r.client == nil {
		return false, errors.New("redis client is nil")
	}

	// 使用 go-redis/v9 推荐的 SetArgs 显式声明 NX 模式
	err := r.client.SetArgs(ctx, key, value, redis.SetArgs{
		Mode: "NX",
		TTL:  expiration,
	}).Err()

	if err == redis.Nil {
		return false, nil // 键已存在，设置失败 (未发生错误，只是没抢到锁)
	}
	if err != nil {
		return false, err // 发生其他网络或底层错误
	}
	return true, nil // 成功设置
}

func (r *RedisAdapter) Eval(ctx context.Context, script string, keys []string, args ...interface{}) error {
	if r.client == nil {
		return errors.New("redis client is nil")
	}
	return r.client.Eval(ctx, script, keys, args...).Err()
}

func (r *RedisAdapter) DeletePrefix(ctx context.Context, prefix string) error {
	if r.client == nil {
		return errors.New("redis client is nil")
	}

	var cursor uint64
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := r.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		if nextCursor == 0 {
			return nil
		}
		cursor = nextCursor
	}
}

// OSSAdapter 将 ossutil 包函数适配为 ObjectStorage 接口
type OSSAdapter struct{}

func NewOSSAdapter() *OSSAdapter {
	return &OSSAdapter{}
}

func (o *OSSAdapter) GenerateSignedURL(objectKey string, expireSec int64) (string, error) {
	return ossutil.GenerateSignedURL(objectKey, expireSec)
}

func (o *OSSAdapter) GeneratePresignedPutURL(objectKey string, expireSec int64) (string, error) {
	return ossutil.GeneratePresignedPutURL(objectKey, expireSec)
}

func (o *OSSAdapter) GenerateCoverKey(courseID uint64, ext string) string {
	return ossutil.GenerateCoverKey(courseID, ext)
}

func (o *OSSAdapter) GenerateVideoKey(courseID uint64, filename string) string {
	return ossutil.GenerateVideoKey(courseID, filename)
}

func (o *OSSAdapter) InitiateMultipartUpload(objectKey string) (string, error) {
	return ossutil.InitiateMultipartUpload(objectKey)
}

func (o *OSSAdapter) GeneratePresignedPartURL(objectKey, uploadID string, partNumber int) (string, error) {
	return ossutil.GeneratePresignedPartURL(objectKey, uploadID, partNumber)
}

func (o *OSSAdapter) CompleteMultipartUpload(objectKey, uploadID string, parts []UploadPart) error {
	realParts := make([]ossutil.UploadPart, len(parts))
	for i, p := range parts {
		realParts[i] = ossutil.UploadPart{
			PartNumber: p.PartNumber,
			ETag:       p.ETag,
		}
	}
	return ossutil.CompleteMultipartUpload(objectKey, uploadID, realParts)
}

// BcryptPasswordManager 密码能力默认实现
type BcryptPasswordManager struct{}

func NewBcryptPasswordManager() *BcryptPasswordManager {
	return &BcryptPasswordManager{}
}

func (b *BcryptPasswordManager) Hash(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func (b *BcryptPasswordManager) Compare(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// JWTTokenProvider Token 能力默认实现
type JWTTokenProvider struct{}

func NewJWTTokenProvider() *JWTTokenProvider {
	return &JWTTokenProvider{}
}

func (j *JWTTokenProvider) Generate(userID int64, role int32) (string, error) {
	return jwt.GenerateToken(userID, role)
}
