package observability

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"geekedu/common/logger"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func UnaryServerTraceInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		traceID := firstMetadataValue(ctx, TraceMetadataKey)
		if traceID == "" {
			traceID = GenerateTraceID()
		}
		ctx = WithTraceID(ctx, traceID)
		return handler(ctx, req)
	}
}

func UnaryServerLoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		service, method := splitMethod(info.FullMethod)
		code := status.Code(err)

		fields := Fields(ctx,
			zap.String("grpc_service", service),
			zap.String("grpc_method", method),
			zap.String("code", code.String()),
			zap.Duration("latency", time.Since(start)),
		)
		if code != codes.OK {
			logger.Log.Warn("gRPC Server Request", fields...)
		} else if GRPCAccessLogEnabled() {
			logger.Log.Info("gRPC Server Request", fields...)
		}
		return resp, err
	}
}

func UnaryServerMetricsInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		service, method := splitMethod(info.FullMethod)
		ObserveGRPCServer(service, method, status.Code(err).String(), time.Since(start))
		return resp, err
	}
}

func UnaryServerRecoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Log.Error("gRPC Server Panic",
					Fields(ctx,
						zap.String("grpc_method", info.FullMethod),
						zap.Any("panic", recovered),
					)...,
				)
				err = status.Error(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

func UnaryClientTraceInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx, traceID := EnsureTraceID(ctx)
		md, _ := metadata.FromOutgoingContext(ctx)
		md = md.Copy()
		md.Set(TraceMetadataKey, traceID)
		ctx = metadata.NewOutgoingContext(ctx, md)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func UnaryClientLoggingInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		service, methodName := splitMethod(method)
		code := status.Code(err)

		fields := Fields(ctx,
			zap.String("target", cc.Target()),
			zap.String("grpc_service", service),
			zap.String("grpc_method", methodName),
			zap.String("code", code.String()),
			zap.Duration("latency", time.Since(start)),
		)
		if code != codes.OK {
			logger.Log.Warn("gRPC Client Request", fields...)
		} else if GRPCAccessLogEnabled() {
			logger.Log.Info("gRPC Client Request", fields...)
		}
		return err
	}
}

func UnaryClientMetricsInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		service, methodName := splitMethod(method)
		ObserveGRPCClient(service, methodName, status.Code(err).String(), time.Since(start))
		return err
	}
}

func splitMethod(fullMethod string) (string, string) {
	service := strings.TrimPrefix(path.Dir(fullMethod), "/")
	method := path.Base(fullMethod)
	return service, method
}

func firstMetadataValue(ctx context.Context, key string) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func NewTraceError(message string) error {
	return fmt.Errorf("%s", message)
}
