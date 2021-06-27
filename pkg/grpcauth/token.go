package grpcauth

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const authKey = "authorization"

func NewServerInterceptor(key string) *ServerTokenInterceptor {
	return &ServerTokenInterceptor{
		key: key,
	}
}

type ServerTokenInterceptor struct {
	key string
}

func (ti *ServerTokenInterceptor) authorize(ctx context.Context, _ string) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "metadata is not provided")
	}
	values := md[authKey]
	if len(values) != 1 {
		return status.Errorf(codes.Unauthenticated, "authorization token is not provided")
	}
	if ti.key != values[0] {
		return status.Errorf(codes.Unauthenticated, "invalid token provided")
	}
	return nil
}

func (ti *ServerTokenInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if err := ti.authorize(ctx, info.FullMethod); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func (ti *ServerTokenInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if err := ti.authorize(stream.Context(), info.FullMethod); err != nil {
			return err
		}
		return handler(srv, stream)
	}
}

func NewClientInterceptor(key string) *ClientTokenInterceptor {
	return &ClientTokenInterceptor{token: key}
}

type ClientTokenInterceptor struct {
	token string
}

func (interceptor *ClientTokenInterceptor) attachToken(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, authKey, interceptor.token)
}

func (interceptor *ClientTokenInterceptor) Unary() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		return invoker(interceptor.attachToken(ctx), method, req, reply, cc, opts...)
	}
}

func (interceptor *ClientTokenInterceptor) Stream() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		return streamer(interceptor.attachToken(ctx), desc, cc, method, opts...)
	}
}
