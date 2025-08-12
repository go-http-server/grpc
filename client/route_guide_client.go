package client

import (
	"context"
	"fmt"
	"io"
	"log"
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

func (rgCli *RouteGuideClient) ListFeatures(rectangle *protoc.Rectangle) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := rgCli.service.ListFeatures(ctx, rectangle)
	if err != nil {
		return err
	}

	for {
		// infinite loop to receive features from stream server
		feature, err := stream.Recv()
		if err == io.EOF {
			// end of the stream
			break
		}

		if err != nil {
			return err
		}

		log.Printf("Feature: name: %q, point:(%v, %v)", feature.GetName(),
			feature.GetLocation().GetLatitude(), feature.GetLocation().GetLongitude())
	}

	return nil
}

func (rgCli *RouteGuideClient) RecordRoute(points []*protoc.Point) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := rgCli.service.RecordRoute(ctx)
	if err != nil {
		return fmt.Errorf("failed to record route: %s, %s", err, stream.RecvMsg(nil))
	}

	for _, point := range points {
		err = stream.Send(point)
		if err != nil {
			return fmt.Errorf("failed send point to server: %s, %s", err, stream.RecvMsg(nil))
		}
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		return err
	}

	log.Printf("record route summary: %+v", res)
	return nil
}
