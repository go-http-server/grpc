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
	"os"
	"os/signal"
	"syscall"

	"aidanwoods.dev/go-paseto"
	"buf.build/go/protovalidate"
	"github.com/go-http-server/grpc/protoc"
	"github.com/go-http-server/grpc/service"
	protovalidate_middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/protovalidate"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip" // gzip compression
	"google.golang.org/grpc/reflection"
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
	flag.Parse()

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

	// configure gRPC server options, enabling authentication and optionally TLS
	grpcServerOpts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			protovalidate_middleware.UnaryServerInterceptor(validator),
			authInterceptor.Unary(),
		),
		grpc.ChainStreamInterceptor(
			protovalidate_middleware.StreamServerInterceptor(validator),
			authInterceptor.Stream(),
		),
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
