package service

import (
	"context"
	"time"

	"github.com/go-http-server/grpc/protoc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthServer is the server API for AuthService service.
type AuthServer struct {
	protoc.UnimplementedAuthServiceServer
	store AccountStore
	maker TokenMaker
}

// NewAuthServer creates a new instance of AuthServer.
func NewAuthServer(store AccountStore, maker TokenMaker) *AuthServer {
	return &AuthServer{store: store, maker: maker}
}

func (s *AuthServer) Login(ctx context.Context, req *protoc.LoginRequest) (*protoc.LoginResponse, error) {
	acc, err := s.store.Find(req.GetUsername())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "store: %s", err)
	}

	err = acc.IsCorrectPassword(req.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "password is incorrect: %s", err)
	}

	token, err := s.maker.CreateToken(acc, 5*time.Minute)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "camnnot create token: %s", err)
	}

	res := &protoc.LoginResponse{
		AccessToken: token,
	}

	return res, nil
}
