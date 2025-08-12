package service

import (
	"context"
	"encoding/json"
	"os"

	"github.com/go-http-server/grpc/protoc"
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

func NewRouteGuideServer() (*RouteGuideServer, error) {
	s := &RouteGuideServer{}

	err := s.loadFeatures("../sample/route_guide_db.json")
	if err != nil {
		return nil, err
	}
	return s, nil
}
