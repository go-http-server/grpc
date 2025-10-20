/*
*
* gRPC is one frame work for remote procedure call (RPC)
* uses HTTP/2 for transport,
* Protocol Buffers as the interface description language, and
* provides features such as
* authentication,
* load balancing, and more.
*
* TLS is transport layer security, it is used to encrypt the data between client and server
* Keepalive is used to check if the connection is still alive
* Graceful shutdown is used to shutdown the server gracefully
* Reflection is used to get the list of services and methods
* Proto validation is used to validate the request and response
* Gzip compression is used to compress the data between client and server
*
 */

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aidanwoods.dev/go-paseto"
	"buf.build/go/protovalidate"
	"github.com/go-http-server/grpc/protoc"
	"github.com/go-http-server/grpc/service"
	protovalidate_middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/protovalidate"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip" // gzip compression
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltracing "google.golang.org/grpc/experimental/opentelemetry"
	otelgrpc "google.golang.org/grpc/stats/opentelemetry"
)

func createAccount(accStore service.AccountStore, username, password, role string) error {
	user, err := service.NewAccount(username, password, role)
	if err != nil {
		return err
	}

	return accStore.Save(user)
}

func seedAccounts(accStore service.AccountStore) error {
	err := createAccount(accStore, "admin_valid", "password", "admin")
	if err != nil {
		return err
	}

	return createAccount(accStore, "user", "password", "user")
}

func accessableRoles() map[string][]string {
	const laptopServiceMethod = "/LaptopService/"
	const routeGuideServiceMethod = "/RouteGuide/"
	return map[string][]string{
		laptopServiceMethod + "CreateLaptop":     {"admin"},
		laptopServiceMethod + "RateLaptop":       {"admin", "user"},
		laptopServiceMethod + "UploadImage":      {"admin"},
		routeGuideServiceMethod + "GetFeature":   {"admin", "user"},
		routeGuideServiceMethod + "ListFeatures": {"admin"},
		routeGuideServiceMethod + "RecordRoute":  {"admin", "user"},
	}
}

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	// Load TLS credentials from a file or other source
	// For simplicity, we are returning nil here, which means no TLS is used.
	// In a real application, you would load your TLS certs and keys.
	serverCert, err := tls.LoadX509KeyPair("certs/server.crt", "certs/server.key")
	if err != nil {
		return nil, err
	}

	pemClientCA, err := os.ReadFile("certs/ca.crt")
	if err != nil {
		return nil, nil
	}

	clientCertPool := x509.NewCertPool()
	if !clientCertPool.AppendCertsFromPEM(pemClientCA) {
		return nil, fmt.Errorf("failed to append client CA certificate")
	}

	// create credentials from the loaded certs
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCertPool,
	}
	return credentials.NewTLS(config), nil
}

func main() {
	port := flag.Int("port", 8080, "Port to run the server on")
	enableTLS := flag.Bool("tls", false, "Enable TLS for the server")
	promAddr := flag.String("prometheus_endpoint", ":9464", "the Prometheus exporter endpoint for metrics")
	flag.Parse()

	// configuration open telemetry for grpc server
	// prometheus exporter
	exporter, err := prometheus.New()
	if err != nil {
		log.Fatalf("failed to initialize prometheus exporter %v", err)
	}
	// configure meter provider for metrics
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	// Configure exporter for traces
	traceExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatalf("Failed to create stdouttrace exporter: %v", err)
	}
	r, err := resource.New(context.Background(),
		resource.WithOS(),
		resource.WithHost(),
		resource.WithProcess(),
		resource.WithFromEnv(),
		resource.WithContainer(),
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithAttributes(semconv.ServiceNameKey.String("grpc-server")),
	)
	// tp: trace provider setup
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(r),
	)
	// Configure W3C Trace Context Propagator for traces
	textMapPropagator := propagation.TraceContext{}
	so := otelgrpc.ServerOption(otelgrpc.Options{
		MetricsOptions: otelgrpc.MetricsOptions{
			MeterProvider: mp,
			Metrics: otelgrpc.DefaultMetrics().Add(
				/* https://grpc.io/docs/guides/opentelemetry-metrics/#instruments
				* Per-call : Observe RPCs themselves (for example, latency.)
				*   Client Per-Call (stable, on by default) : Observe a client call
				*   Client Per-Attempt (stable, on by default) : Observe attempts for a client call, since a call can have multiple attempts due to retry or hedging.
				*   Client Per-Call Retry (experimental) : Observe retry, transparent retry and hedging,
				*   Server : Observe a call received at the server.
				* LB Policy : Observe various load-balancing policies
				*   Weighted Round Robin (experimental)
				*   Pick-First (experimental)
				* XdsClient (experimental)
				 */
				// Pick First LB Policy Instruments
				"grpc.lb.pick_first.disconnections",
				"grpc.lb.pick_first.connection_attempts_succeeded",
				"grpc.lb.pick_first.connection_attempts_failed",
				// Weighted Round Robin LB Policy Instruments
				"grpc.lb.wrr.endpoint_weights",
				"grpc.lb.wrr.rr_fallback",
				"grpc.lb.wrr.endpoint_weight_not_yet_usable",
				"grpc.lb.wrr.endpoint_weight_stale",
			),
		},
		TraceOptions: oteltracing.TraceOptions{
			TracerProvider:    tp,
			TextMapPropagator: textMapPropagator,
		},
	})

	// run metrics server in a separate goroutine
	go http.ListenAndServe(*promAddr, promhttp.Handler())

	laptopServer := service.NewLaptopServer(service.NewInMemoryLaptopStore(), service.NewDiskImageStore("images"), service.NewInMemoryRatingStore())
	accountStore := service.NewInMemoryAccountStore()
	tokenMaker := service.NewPasetoMaker(paseto.NewV4AsymmetricSecretKey(), paseto.NewParserWithoutExpiryCheck())
	authServer := service.NewAuthServer(accountStore, tokenMaker)
	routeGuideServer, err := service.NewRouteGuideServer()
	if err != nil {
		log.Fatalf("failed to create route guide server: %v", err)
	}

	authInterceptor := service.NewAuthInterceptor(tokenMaker, accessableRoles())

	validator, err := protovalidate.New(
		protovalidate.WithFailFast(),
		protovalidate.WithMessages(
			&protoc.LoginRequest{}, // make ensures validator has pre-warmed messages
			&protoc.CreateLaptopRequest{},
			&protoc.SearchLaptopRequest{},
			&protoc.RateLaptopRequest{},
			&protoc.UploadImageRequest{},
			&protoc.Point{},
			&protoc.Rectangle{},
			&protoc.RouteNote{},
		),
		// protovalidate.WithMessages(protoc.File_auth_auth_service_proto.Options()), // wrong pre-warn declaration, haven't error runtime, but no effect
	)
	if err != nil {
		log.Fatalf("failed to create validator: %v", err)
	}

	// keepalive grpc server option make keepalive working.
	kaep := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second, // if a client pings more than once every 5 seconds, terminate the connection
		PermitWithoutStream: true,            // allow pings even when there are no active streams
	}
	kasp := keepalive.ServerParameters{
		MaxConnectionIdle:     15 * time.Second, // If a client is idle for 15 seconds, send a GOAWAY
		MaxConnectionAge:      30 * time.Second, // If any connection is alive for more than 30 seconds, send a GOAWAY
		MaxConnectionAgeGrace: 5 * time.Second,  // Allow 5 seconds for pending RPCs to complete before forcibly closing connections
		Time:                  5 * time.Second,  // Ping the client if it is idle for 5 seconds to ensure the connection is still active
		Timeout:               1 * time.Second,  // Wait 1 second for the ping ack before assuming the connection is dead
	}

	// configure gRPC server options, enabling authentication and optionally TLS
	grpcServerOpts := []grpc.ServerOption{
		grpc.KeepaliveEnforcementPolicy(kaep),
		grpc.KeepaliveParams(kasp),
		grpc.ChainUnaryInterceptor(
			protovalidate_middleware.UnaryServerInterceptor(validator),
			authInterceptor.Unary(),
		),
		grpc.ChainStreamInterceptor(
			protovalidate_middleware.StreamServerInterceptor(validator),
			authInterceptor.Stream(),
		),
		so,
	}
	if *enableTLS {
		tlsCredentials, err := loadTLSCredentials()
		if err != nil {
			log.Fatalf("failed to load TLS credentials: %v", err)
		}
		grpcServerOpts = append(grpcServerOpts, grpc.Creds(tlsCredentials))
	}

	// create a new gRPC server with the configured options
	grpcServer := grpc.NewServer(grpcServerOpts...)

	// register the services with the gRPC server
	protoc.RegisterLaptopServiceServer(grpcServer, laptopServer)
	protoc.RegisterAuthServiceServer(grpcServer, authServer)
	protoc.RegisterRouteGuideServer(grpcServer, routeGuideServer)
	reflection.Register(grpcServer)

	err = seedAccounts(accountStore)
	if err != nil {
		log.Fatalf("cannot seed accounts: %s", err)
	}

	addr := fmt.Sprintf("0.0.0.0:%d", *port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("cannot create server on port :%d, err: %s", *port, err)
	}

	// create signal context to handle graceful shutdown from interrupt, sigterm, sisint signals
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// create group to manage server goroutine and shutdown goroutine, ctx serve task handler in worker pool
	gr, ctx := errgroup.WithContext(sigCtx)

	gr.Go(func() error {
		log.Printf("Starting server on port :%d with tls option: %t", *port, *enableTLS)
		err = grpcServer.Serve(listener)
		if err != nil {
			if errors.Is(err, grpc.ErrServerStopped) {
				return nil
			}

			log.Printf("cannot start server on port :%d, err: %s", *port, err)
			return err
		}

		return nil
	})

	gr.Go(func() error {
		<-ctx.Done()
		// implement graceful shutdown
		log.Println("shutting down gRPC server...")
		grpcServer.GracefulStop()
		return nil
	})

	if err := gr.Wait(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
