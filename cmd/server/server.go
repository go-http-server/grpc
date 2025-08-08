package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/go-http-server/grpc/protoc"
	"github.com/go-http-server/grpc/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func unaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	log.Println("--- Unary Interceptor ---", info.FullMethod)

	return handler(ctx, req)
}

func serverStreamInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	log.Println("--- Stream Interceptor ---", info.FullMethod)
	return handler(srv, ss)
}

func main() {
	port := flag.Int("port", 8080, "Port to run the server on")
	flag.Parse()

	laptopServer := service.NewLaptopServer(service.NewInMemoryLaptopStore(), service.NewDiskImageStore("images"), service.NewInMemoryRatingStore())
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(unaryInterceptor),
		grpc.StreamInterceptor(serverStreamInterceptor),
	)
	protoc.RegisterLaptopServiceServer(grpcServer, laptopServer)
	reflection.Register(grpcServer)

	addr := fmt.Sprintf("0.0.0.0:%d", *port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("cannot create server on port :%d, err: %s", *port, err)
	}

	log.Printf("Starting server on port :%d", *port)
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("cannot start server on port :%d, err: %s", *port, err)
	}
}
