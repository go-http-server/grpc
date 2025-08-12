package service

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"os"
	"time"

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
	for _, feat := range s.savedFeatures {
		err := contextError(ctx)
		if err != nil {
			return nil, err
		}
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

func (s *RouteGuideServer) RecordRoute(streaming grpc.ClientStreamingServer[protoc.Point, protoc.RouteSummary]) error {
	var pointCount, featureCount, distance int32
	var lastPoint *protoc.Point
	startTime := time.Now()

	for {
		err := contextError(streaming.Context())
		if err != nil {
			return err
		}

		point, err := streaming.Recv()
		if err == io.EOF {
			// streaming client request is end of life -> return response to client
			endTime := time.Now()
			return streaming.SendAndClose(&protoc.RouteSummary{
				PointCount:   pointCount,
				FeatureCount: featureCount,
				Distance:     distance,
				ElapsedTime:  int32(endTime.Sub(startTime).Seconds()),
			})
		}

		if err != nil {
			return status.Errorf(codes.Internal, "cannot receive streaming request from client: %s", err)
		}

		pointCount++
		for _, feature := range s.savedFeatures {
			if proto.Equal(feature.Location, point) {
				featureCount++
			}
		}

		if lastPoint != nil {
			distance += calcDistance(lastPoint, point)
		}
		lastPoint = point
	}
}

func toRadians(num float64) float64 {
	return num * math.Pi / float64(180)
}

// calcDistance calculates the distance between two points using the "haversine" formula.
// The formula is based on http://mathforum.org/library/drmath/view/51879.html.
func calcDistance(p1 *protoc.Point, p2 *protoc.Point) int32 {
	const CordFactor float64 = 1e7
	const R = float64(6371000) // earth radius in metres
	lat1 := toRadians(float64(p1.Latitude) / CordFactor)
	lat2 := toRadians(float64(p2.Latitude) / CordFactor)
	lng1 := toRadians(float64(p1.Longitude) / CordFactor)
	lng2 := toRadians(float64(p2.Longitude) / CordFactor)
	dlat := lat2 - lat1
	dlng := lng2 - lng1

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(dlng/2)*math.Sin(dlng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := R * c
	return int32(distance)
}
