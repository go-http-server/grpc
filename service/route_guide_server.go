package service

import (
	"context"
	"encoding/json"
	"math"
	"os"

	"github.com/go-http-server/grpc/protoc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// RouteGuideServer implements the RouteGuide service.
type RouteGuideServer struct {
	protoc.UnimplementedRouteGuideServer
	savedFeatures []*protoc.Feature // readonly after initialize
}

// loadFeatures loads features from a JSON file into the server's savedFeatures slice.
func (s *RouteGuideServer) loadFeatures(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &s.savedFeatures)
	if err != nil {
		return err
	}

	return nil
}

// NewRouteGuideServer creates a new instance of RouteGuideServer and loads features from a JSON file.
func NewRouteGuideServer() (*RouteGuideServer, error) {
	s := &RouteGuideServer{}

	err := s.loadFeatures("./sample/route_guide.json")
	if err != nil {
		return nil, err
	}
	return s, nil
}

// GetFeature retrieves the feature at the given point and implements the GetFeature method of the RouteGuideServer interface.
func (s *RouteGuideServer) GetFeature(ctx context.Context, point *protoc.Point) (*protoc.Feature, error) {
	err := contextError(ctx)
	if err != nil {
		return nil, err
	}

	for _, feat := range s.savedFeatures {
		// compare proto messages for equality
		if proto.Equal(feat.Location, point) {
			return feat, nil
		}
	}

	// return point feature if it exists with unnamed
	return &protoc.Feature{Location: point}, nil
}

// inRange checks if a point is within the given rectangle.
func inRange(point *protoc.Point, rect *protoc.Rectangle) bool {
	left := math.Min(float64(rect.Lo.Longitude), float64(rect.Hi.Longitude))
	right := math.Max(float64(rect.Lo.Longitude), float64(rect.Hi.Longitude))
	top := math.Max(float64(rect.Lo.Latitude), float64(rect.Hi.Latitude))
	bottom := math.Min(float64(rect.Lo.Latitude), float64(rect.Hi.Latitude))

	if float64(point.Longitude) >= left &&
		float64(point.Longitude) <= right &&
		float64(point.Latitude) >= bottom &&
		float64(point.Latitude) <= top {
		return true
	}
	return false
}

func (s *RouteGuideServer) ListFeatures(req *protoc.Rectangle, streaming grpc.ServerStreamingServer[protoc.Feature]) error {
	for _, feature := range s.savedFeatures {
		err := contextError(streaming.Context())
		if err != nil {
			return err
		}

		if inRange(feature.Location, req) {
			if err := streaming.Send(feature); err != nil {
				return status.Errorf(codes.Internal, "failed to send feature: %v", err)
			}
		}
	}

	return nil
}
