package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"geekedu/common/logger"
	"geekedu/common/observability"
	"geekedu/common/jwt"
	ossutil "geekedu/logic-server/oss"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var ErrRedisCircuitOpen = errors.New("redis circuit open")

// RedisAdapter 将 redis.Client 适配为 Cache 接口
type RedisAdapter struct {
	client       *redis.Client
	cacheBreaker *redisCircuitBreaker
	lockBreaker  *redisCircuitBreaker
}

func NewRedisAdapter(client *redis.Client) *RedisAdapter {
	return &RedisAdapter{
		client:       client,
		cacheBreaker: newRedisCircuitBreaker("course_cache", 5, 5*time.Second),
		lockBreaker:  newRedisCircuitBreaker("order_lock", 5, 5*time.Second),
	}
}

func (r *RedisAdapter) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if r.client == nil {
		observability.ObserveCacheRequest("course_cache", "get", "error")
		return nil, false, errors.New("redis client is nil")
	}
	if err := r.cacheBreaker.beforeRequest(); err != nil {
		observability.ObserveCacheRequest("course_cache", "get", "fast_fail")
		return nil, false, err
	}
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		r.cacheBreaker.afterSuccess()
		observability.ObserveCacheRequest("course_cache", "get", "miss")
		return nil, false, nil
	}
	if err != nil {
		r.cacheBreaker.afterFailure(err)
		observability.ObserveCacheRequest("course_cache", "get", "error")
		return nil, false, err
	}
	r.cacheBreaker.afterSuccess()
	observability.ObserveCacheRequest("course_cache", "get", "hit")
	return []byte(val), true, nil
}

func (r *RedisAdapter) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	if r.client == nil {
		observability.ObserveCacheRequest("course_cache", "set", "error")
		return errors.New("redis client is nil")
	}
	if err := r.cacheBreaker.beforeRequest(); err != nil {
		observability.ObserveCacheRequest("course_cache", "set", "fast_fail")
		return err
	}
	if err := r.client.Set(ctx, key, value, expiration).Err(); err != nil {
		r.cacheBreaker.afterFailure(err)
		observability.ObserveCacheRequest("course_cache", "set", "error")
		return err
	}
	r.cacheBreaker.afterSuccess()
	observability.ObserveCacheRequest("course_cache", "set", "ok")
	return nil
}

func (r *RedisAdapter) SetNX(ctx context.Context, key string, value string, expiration time.Duration) (bool, error) {
	if r.client == nil {
		observability.ObserveCacheRequest("order_lock", "setnx", "error")
		return false, errors.New("redis client is nil")
	}
	if err := r.lockBreaker.beforeRequest(); err != nil {
		observability.ObserveCacheRequest("order_lock", "setnx", "fast_fail")
		return false, err
	}

	// 使用 go-redis/v9 推荐的 SetArgs 显式声明 NX 模式
	err := r.client.SetArgs(ctx, key, value, redis.SetArgs{
		Mode: "NX",
		TTL:  expiration,
	}).Err()

	if err == redis.Nil {
		r.lockBreaker.afterSuccess()
		observability.ObserveCacheRequest("order_lock", "setnx", "miss")
		return false, nil // 键已存在，设置失败 (未发生错误，只是没抢到锁)
	}
	if err != nil {
		r.lockBreaker.afterFailure(err)
		observability.ObserveCacheRequest("order_lock", "setnx", "error")
		return false, err // 发生其他网络或底层错误
	}
	r.lockBreaker.afterSuccess()
	observability.ObserveCacheRequest("order_lock", "setnx", "ok")
	return true, nil // 成功设置
}

func (r *RedisAdapter) Eval(ctx context.Context, script string, keys []string, args ...interface{}) error {
	if r.client == nil {
		observability.ObserveCacheRequest("order_lock", "eval", "error")
		return errors.New("redis client is nil")
	}
	if err := r.lockBreaker.beforeRequest(); err != nil {
		observability.ObserveCacheRequest("order_lock", "eval", "fast_fail")
		return err
	}
	if err := r.client.Eval(ctx, script, keys, args...).Err(); err != nil {
		r.lockBreaker.afterFailure(err)
		observability.ObserveCacheRequest("order_lock", "eval", "error")
		return err
	}
	r.lockBreaker.afterSuccess()
	observability.ObserveCacheRequest("order_lock", "eval", "ok")
	return nil
}

func (r *RedisAdapter) DeletePrefix(ctx context.Context, prefix string) error {
	if r.client == nil {
		observability.ObserveCacheRequest("course_cache", "delete_prefix", "error")
		return errors.New("redis client is nil")
	}
	if err := r.cacheBreaker.beforeRequest(); err != nil {
		observability.ObserveCacheRequest("course_cache", "delete_prefix", "fast_fail")
		return err
	}

	var cursor uint64
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			r.cacheBreaker.afterFailure(err)
			observability.ObserveCacheRequest("course_cache", "delete_prefix", "error")
			return err
		}
		if len(keys) > 0 {
			if err := r.client.Del(ctx, keys...).Err(); err != nil {
				r.cacheBreaker.afterFailure(err)
				observability.ObserveCacheRequest("course_cache", "delete_prefix", "error")
				return err
			}
		}
		if nextCursor == 0 {
			r.cacheBreaker.afterSuccess()
			observability.ObserveCacheRequest("course_cache", "delete_prefix", "ok")
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

type circuitState string

const (
	circuitClosed   circuitState = "closed"
	circuitOpen     circuitState = "open"
	circuitHalfOpen circuitState = "half_open"
)

type redisCircuitBreaker struct {
	component        string
	failureThreshold int
	openDuration     time.Duration

	mu            sync.Mutex
	state         circuitState
	failures      int
	openUntil     time.Time
	probeInFlight bool
}

func newRedisCircuitBreaker(component string, failureThreshold int, openDuration time.Duration) *redisCircuitBreaker {
	observability.SetRedisCircuitOpen(component, false)
	return &redisCircuitBreaker{
		component:        component,
		failureThreshold: failureThreshold,
		openDuration:     openDuration,
		state:            circuitClosed,
	}
}

func (b *redisCircuitBreaker) beforeRequest() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	switch b.state {
	case circuitClosed:
		return nil
	case circuitOpen:
		if now.Before(b.openUntil) {
			return ErrRedisCircuitOpen
		}
		b.state = circuitHalfOpen
		b.probeInFlight = true
		return nil
	case circuitHalfOpen:
		if b.probeInFlight {
			return ErrRedisCircuitOpen
		}
		b.probeInFlight = true
		return nil
	default:
		return nil
	}
}

func (b *redisCircuitBreaker) afterSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	wasOpen := b.state == circuitOpen || b.state == circuitHalfOpen
	b.state = circuitClosed
	b.failures = 0
	b.openUntil = time.Time{}
	b.probeInFlight = false
	observability.SetRedisCircuitOpen(b.component, false)
	if wasOpen {
		getLogger().Info("Redis circuit recovered", zap.String("component", b.component))
	}
}

func (b *redisCircuitBreaker) afterFailure(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state == circuitHalfOpen {
		b.tripLocked("Redis circuit half-open probe failed", err)
		return
	}

	b.failures++
	b.probeInFlight = false
	if b.failures >= b.failureThreshold && b.state != circuitOpen {
		b.tripLocked("Redis circuit opened", err)
	}
}

func (b *redisCircuitBreaker) tripLocked(message string, err error) {
	b.state = circuitOpen
	b.failures = 0
	b.openUntil = time.Now().Add(b.openDuration)
	b.probeInFlight = false
	observability.SetRedisCircuitOpen(b.component, true)
	observability.IncRedisCircuitOpen(b.component)
	getLogger().Warn(message,
		zap.String("component", b.component),
		zap.Duration("open_duration", b.openDuration),
		zap.Error(err),
	)
}

func getLogger() *zap.Logger {
	if logger.Log != nil {
		return logger.Log
	}
	return zap.NewNop()
}
