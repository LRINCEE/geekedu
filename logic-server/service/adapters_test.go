package service

import (
	"errors"
	"testing"
	"time"
)

func TestRedisCircuitBreakerOpensAndFastFails(t *testing.T) {
	breaker := newRedisCircuitBreaker("course_cache", 2, 50*time.Millisecond)

	if err := breaker.beforeRequest(); err != nil {
		t.Fatalf("expected first request to pass, got %v", err)
	}
	breaker.afterFailure(errors.New("redis down"))

	if err := breaker.beforeRequest(); err != nil {
		t.Fatalf("expected second request to pass before threshold, got %v", err)
	}
	breaker.afterFailure(errors.New("redis still down"))

	if err := breaker.beforeRequest(); !errors.Is(err, ErrRedisCircuitOpen) {
		t.Fatalf("expected fast fail after circuit opens, got %v", err)
	}
}

func TestRedisCircuitBreakerRecoversAfterHalfOpenProbeSuccess(t *testing.T) {
	breaker := newRedisCircuitBreaker("course_cache", 1, 10*time.Millisecond)

	if err := breaker.beforeRequest(); err != nil {
		t.Fatalf("expected request to pass, got %v", err)
	}
	breaker.afterFailure(errors.New("redis down"))

	time.Sleep(20 * time.Millisecond)

	if err := breaker.beforeRequest(); err != nil {
		t.Fatalf("expected half-open probe to be allowed, got %v", err)
	}
	breaker.afterSuccess()

	if err := breaker.beforeRequest(); err != nil {
		t.Fatalf("expected breaker to recover to closed state, got %v", err)
	}
}
