package service

import (
	"context"
	"log"
	"slices"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthInterceptor is a middleware that checks if the user is authenticated and has the required role to access the endpoint.
type AuthInterceptor struct {
	maker           TokenMaker
	accessableRoles map[string][]string
}

// NewAuthInterceptor creates a new AuthInterceptor with the given TokenMaker and accessable roles.
func NewAuthInterceptor(maker TokenMaker, accessableRoles map[string][]string) *AuthInterceptor {
	return &AuthInterceptor{maker: maker, accessableRoles: accessableRoles}
}

// Unary returns a unary server interceptor that checks if the user is authenticated and has the required role to access the endpoint.
func (interceptor *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		log.Println("--- Unary Interceptor ---", info.FullMethod)

		err := interceptor.authorize(ctx, info.FullMethod)
		if err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

// Stream returns a stream server interceptor that checks if the user is authenticated and has the required role to access the endpoint.
func (interceptor *AuthInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		log.Println("--- Stream Interceptor ---", info.FullMethod)

		err := interceptor.authorize(ss.Context(), info.FullMethod)
		if err != nil {
			return err
		}

		return handler(srv, ss)
	}
}

func (interceptor *AuthInterceptor) authorize(ctx context.Context, method string) error {
	accessableRoles, ok := interceptor.accessableRoles[method]
	if !ok {
		// public method, no authorization needed
		return nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "metadata not provided")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return status.Errorf(codes.Unauthenticated, "authorization token not provided")
	}

	accessToken := values[0]
	payload, err := interceptor.maker.VerifyToken(accessToken)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "invalid access token: %v", err)
	}

	if slices.Contains(accessableRoles, payload.Role) {
		return nil
	}

	return status.Errorf(codes.PermissionDenied, "user with role %s is not allowed to access %s", payload.Role, method)
}
