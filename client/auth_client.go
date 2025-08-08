// Package client provides the client-side implementation for authentication services.
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/go-http-server/grpc/protoc"
	"google.golang.org/grpc"
)

// AuthClient is a client for interacting with the authentication service.
type AuthClient struct {
	service            protoc.AuthServiceClient
	username, password string
}

// NewAuthClient creates a new AuthClient instance.
func NewAuthClient(cc *grpc.ClientConn, username, password string) *AuthClient {
	service := protoc.NewAuthServiceClient(cc)
	return &AuthClient{service: service, username: username, password: password}
}

func (client *AuthClient) Login() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	req := &protoc.LoginRequest{
		Username: client.username,
		Password: client.password,
	}

	res, err := client.service.Login(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to login: %s", err)
	}

	return res.GetAccessToken(), nil
}
