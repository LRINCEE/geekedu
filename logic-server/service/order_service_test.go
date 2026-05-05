package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"geekedu/common/pb"
	"geekedu/logic-server/model"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type mockOrderRepo struct {
	mu          sync.Mutex
	orders      []*model.Order
	purchaseMap map[uint64]map[uint64]bool
	createErr   error
	checkErr    error
}

func (m *mockOrderRepo) CreateOrder(order *model.Order) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.createErr != nil {
		return m.createErr
	}
	order.ID = uint64(len(m.orders) + 2001)
	m.orders = append(m.orders, order)
	return nil
}

func (m *mockOrderRepo) CheckPurchase(userID, courseID uint64) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.checkErr != nil {
		return false, m.checkErr
	}
	return m.purchaseMap[userID][courseID], nil
}

type mockCourseRepoForOrder struct {
	courses map[uint64]*model.Course
}

func newMockCourseRepoForOrder() *mockCourseRepoForOrder {
	return &mockCourseRepoForOrder{
		courses: map[uint64]*model.Course{
			101: {ID: 101, Title: "Go microservice", Price: 99.0},
			102: {ID: 102, Title: "Redis deep dive", Price: 199.0},
		},
	}
}

func (m *mockCourseRepoForOrder) GetCourseByID(id uint64) (*model.Course, error) {
	c, ok := m.courses[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return c, nil
}

func (m *mockCourseRepoForOrder) CreateCourse(course *model.Course) error { return nil }

func (m *mockCourseRepoForOrder) ListCourses(page, pageSize int) ([]*model.Course, int64, error) {
	return nil, 0, nil
}

func (m *mockCourseRepoForOrder) UpdateCoverKey(id uint64, coverKey string) error { return nil }

type mockCache struct {
	mu          sync.Mutex
	locks       map[string]string
	setNXResult bool
	setNXErr    error
	evalErr     error
}

func newMockCache() *mockCache {
	return &mockCache{locks: make(map[string]string)}
}

func (m *mockCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	val, ok := m.locks[key]
	if !ok {
		return nil, false, nil
	}
	return []byte(val), true, nil
}

func (m *mockCache) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.locks[key] = string(value)
	return nil
}

func (m *mockCache) SetNX(ctx context.Context, key string, value string, expiration time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.setNXErr != nil {
		return false, m.setNXErr
	}
	if _, exists := m.locks[key]; exists {
		return false, nil
	}
	m.locks[key] = value
	return m.setNXResult, nil
}

func (m *mockCache) Eval(ctx context.Context, script string, keys []string, args ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.evalErr != nil {
		return m.evalErr
	}
	delete(m.locks, keys[0])
	return nil
}

func TestOrderService_CreateOrder(t *testing.T) {
	tests := []struct {
		name         string
		req          *pb.CreateOrderRequest
		setupRepos   func() (*mockOrderRepo, *mockCourseRepoForOrder, *mockCache)
		wantErr      bool
		wantGRPCCode codes.Code
		wantOrderID  int64
	}{
		{
			name: "success creates order",
			req:  &pb.CreateOrderRequest{UserId: 1, CourseId: 101},
			setupRepos: func() (*mockOrderRepo, *mockCourseRepoForOrder, *mockCache) {
				cache := newMockCache()
				cache.setNXResult = true
				return &mockOrderRepo{purchaseMap: map[uint64]map[uint64]bool{1: {}}}, newMockCourseRepoForOrder(), cache
			},
			wantOrderID: 2001,
		},
		{
			name: "already purchased returns AlreadyExists",
			req:  &pb.CreateOrderRequest{UserId: 1, CourseId: 101},
			setupRepos: func() (*mockOrderRepo, *mockCourseRepoForOrder, *mockCache) {
				cache := newMockCache()
				cache.setNXResult = true
				return &mockOrderRepo{purchaseMap: map[uint64]map[uint64]bool{1: {101: true}}}, newMockCourseRepoForOrder(), cache
			},
			wantErr:      true,
			wantGRPCCode: codes.AlreadyExists,
		},
		{
			name: "missing course returns NotFound",
			req:  &pb.CreateOrderRequest{UserId: 1, CourseId: 9999},
			setupRepos: func() (*mockOrderRepo, *mockCourseRepoForOrder, *mockCache) {
				cache := newMockCache()
				cache.setNXResult = true
				return &mockOrderRepo{purchaseMap: map[uint64]map[uint64]bool{1: {}}}, &mockCourseRepoForOrder{courses: map[uint64]*model.Course{}}, cache
			},
			wantErr:      true,
			wantGRPCCode: codes.NotFound,
		},
		{
			name: "lock conflict returns Aborted",
			req:  &pb.CreateOrderRequest{UserId: 1, CourseId: 101},
			setupRepos: func() (*mockOrderRepo, *mockCourseRepoForOrder, *mockCache) {
				cache := newMockCache()
				cache.setNXResult = false
				return &mockOrderRepo{purchaseMap: map[uint64]map[uint64]bool{1: {}}}, newMockCourseRepoForOrder(), cache
			},
			wantErr:      true,
			wantGRPCCode: codes.Aborted,
		},
		{
			name: "redis SetNX failure falls back to database guard",
			req:  &pb.CreateOrderRequest{UserId: 1, CourseId: 101},
			setupRepos: func() (*mockOrderRepo, *mockCourseRepoForOrder, *mockCache) {
				cache := newMockCache()
				cache.setNXErr = errors.New("redis connection refused")
				return &mockOrderRepo{purchaseMap: map[uint64]map[uint64]bool{1: {}}}, newMockCourseRepoForOrder(), cache
			},
			wantOrderID: 2001,
		},
		{
			name: "purchase check db error returns Internal",
			req:  &pb.CreateOrderRequest{UserId: 1, CourseId: 101},
			setupRepos: func() (*mockOrderRepo, *mockCourseRepoForOrder, *mockCache) {
				cache := newMockCache()
				cache.setNXResult = true
				return &mockOrderRepo{checkErr: errors.New("db error")}, newMockCourseRepoForOrder(), cache
			},
			wantErr:      true,
			wantGRPCCode: codes.Internal,
		},
		{
			name: "create order db error returns Internal",
			req:  &pb.CreateOrderRequest{UserId: 1, CourseId: 101},
			setupRepos: func() (*mockOrderRepo, *mockCourseRepoForOrder, *mockCache) {
				cache := newMockCache()
				cache.setNXResult = true
				return &mockOrderRepo{purchaseMap: map[uint64]map[uint64]bool{1: {}}, createErr: errors.New("db error")}, newMockCourseRepoForOrder(), cache
			},
			wantErr:      true,
			wantGRPCCode: codes.Internal,
		},
		{
			name: "unlock error does not block success",
			req:  &pb.CreateOrderRequest{UserId: 1, CourseId: 101},
			setupRepos: func() (*mockOrderRepo, *mockCourseRepoForOrder, *mockCache) {
				cache := newMockCache()
				cache.setNXResult = true
				cache.evalErr = errors.New("eval error")
				return &mockOrderRepo{purchaseMap: map[uint64]map[uint64]bool{1: {}}}, newMockCourseRepoForOrder(), cache
			},
			wantOrderID: 2001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo, courseRepo, cache := tt.setupRepos()
			svc := NewOrderServiceServer(orderRepo, courseRepo, cache)

			resp, err := svc.CreateOrder(context.Background(), tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantGRPCCode, st.Code(), "expected gRPC code %s, got %s", tt.wantGRPCCode, st.Code())
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantOrderID, resp.OrderId)
		})
	}
}

func TestOrderService_CreateOrder_ConcurrentSafety(t *testing.T) {
	cache := newMockCache()
	cache.setNXResult = true

	orderRepo := &mockOrderRepo{purchaseMap: map[uint64]map[uint64]bool{1: {}}}
	courseRepo := newMockCourseRepoForOrder()
	svc := NewOrderServiceServer(orderRepo, courseRepo, cache)

	const concurrency = 10
	results := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			_, err := svc.CreateOrder(context.Background(), &pb.CreateOrderRequest{UserId: 1, CourseId: 101})
			results <- err
		}()
	}

	var successCount, failCount int
	for i := 0; i < concurrency; i++ {
		if err := <-results; err == nil {
			successCount++
		} else {
			failCount++
		}
	}

	assert.Equal(t, concurrency, successCount+failCount)
	assert.GreaterOrEqual(t, successCount, 1)
}

func TestOrderService_CheckPurchase(t *testing.T) {
	orderRepo := &mockOrderRepo{
		purchaseMap: map[uint64]map[uint64]bool{1: {101: true, 102: false}},
	}
	courseRepo := &mockCourseRepoForOrder{}
	cache := newMockCache()
	svc := NewOrderServiceServer(orderRepo, courseRepo, cache)

	t.Run("purchased course returns true", func(t *testing.T) {
		resp, err := svc.CheckPurchase(context.Background(), &pb.CheckPurchaseRequest{UserId: 1, CourseId: 101})
		assert.NoError(t, err)
		assert.True(t, resp.Purchased)
	})

	t.Run("unpurchased course returns false", func(t *testing.T) {
		resp, err := svc.CheckPurchase(context.Background(), &pb.CheckPurchaseRequest{UserId: 1, CourseId: 102})
		assert.NoError(t, err)
		assert.False(t, resp.Purchased)
	})
}
