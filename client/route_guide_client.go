package client

import (
	"context"
	"time"

	"github.com/go-http-server/grpc/protoc"
	"google.golang.org/grpc"
)

// RouteGuideClient is a client for interacting with the RouteGuide service.
type RouteGuideClient struct {
	service protoc.RouteGuideClient
}

func NewRouteGuideClient(cc *grpc.ClientConn) *RouteGuideClient {
	service := protoc.NewRouteGuideClient(cc)
	return &RouteGuideClient{service: service}
}

func (rgCli *RouteGuideClient) GetFeature(point *protoc.Point) (*protoc.Feature, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	feature, err := rgCli.service.GetFeature(ctx, point)
	if err != nil {
		return nil, err
	}

	return feature, nil
}
