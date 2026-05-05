package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestTraceMiddlewareReusesExistingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceMiddleware())
	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, TraceIDFromContext(c.Request.Context()))
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set(TraceHeader, "trace-from-client")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if got := w.Header().Get(TraceHeader); got != "trace-from-client" {
		t.Fatalf("expected response trace header to be reused, got %q", got)
	}
	if got := w.Body.String(); got != "trace-from-client" {
		t.Fatalf("expected request context trace id to be reused, got %q", got)
	}
}

func TestTraceMiddlewareGeneratesHeaderWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceMiddleware())
	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if got := w.Header().Get(TraceHeader); got == "" {
		t.Fatal("expected response trace header to be generated")
	}
}

func TestUnaryClientTraceInterceptorInjectsMetadata(t *testing.T) {
	interceptor := UnaryClientTraceInterceptor()
	ctx := WithTraceID(context.Background(), "client-trace-id")

	err := interceptor(ctx, "/course.CourseService/ListCourses", nil, nil, &grpc.ClientConn{}, func(
		callCtx context.Context,
		method string,
		req any,
		reply any,
		cc *grpc.ClientConn,
		opts ...grpc.CallOption,
	) error {
		md, ok := metadata.FromOutgoingContext(callCtx)
		if !ok {
			t.Fatal("expected outgoing metadata to exist")
		}
		values := md.Get(TraceMetadataKey)
		if len(values) != 1 || values[0] != "client-trace-id" {
			t.Fatalf("expected outgoing trace metadata, got %#v", values)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected interceptor error: %v", err)
	}
}

func TestUnaryServerTraceInterceptorLoadsMetadataIntoContext(t *testing.T) {
	interceptor := UnaryServerTraceInterceptor()
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(TraceMetadataKey, "server-trace-id"))

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/course.CourseService/ListCourses"}, func(callCtx context.Context, req any) (any, error) {
		if got := TraceIDFromContext(callCtx); got != "server-trace-id" {
			t.Fatalf("expected trace id in server context, got %q", got)
		}
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected interceptor error: %v", err)
	}
}
