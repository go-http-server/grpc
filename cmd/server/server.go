package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"

	"aidanwoods.dev/go-paseto"
	"github.com/go-http-server/grpc/protoc"
	"github.com/go-http-server/grpc/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
	err := createAccount(accStore, "admin", "password", "admin")
	if err != nil {
		return err
	}

	return createAccount(accStore, "user", "password", "user")
}

func accessableRoles() map[string][]string {
	const laptopServiceMethod = "/LaptopService/"
	return map[string][]string{
		laptopServiceMethod + "CreateLaptop": {"admin"},
		laptopServiceMethod + "SearchLaptop": {"admin", "user"},
		laptopServiceMethod + "RateLaptop":   {"admin", "user"},
		laptopServiceMethod + "UploadImage":  {"admin"},
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

	// create credentials from the loaded certs
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.NoClientCert,
	}
	return credentials.NewTLS(config), nil
}

func main() {
	port := flag.Int("port", 8080, "Port to run the server on")
	flag.Parse()

	laptopServer := service.NewLaptopServer(service.NewInMemoryLaptopStore(), service.NewDiskImageStore("images"), service.NewInMemoryRatingStore())
	accountStore := service.NewInMemoryAccountStore()
	tokenMaker := service.NewPasetoMaker(paseto.NewV4AsymmetricSecretKey(), paseto.NewParserWithoutExpiryCheck())
	authServer := service.NewAuthServer(accountStore, tokenMaker)

	authInterceptor := service.NewAuthInterceptor(tokenMaker, accessableRoles())

	tlsCredentials, err := loadTLSCredentials()
	if err != nil {
		log.Fatalf("failed to load TLS credentials: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.Creds(tlsCredentials),
		grpc.UnaryInterceptor(authInterceptor.Unary()),
		grpc.StreamInterceptor(authInterceptor.Stream()),
	)
	protoc.RegisterLaptopServiceServer(grpcServer, laptopServer)
	protoc.RegisterAuthServiceServer(grpcServer, authServer)
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

	log.Printf("Starting server on port :%d", *port)
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("cannot start server on port :%d, err: %s", *port, err)
	}
}
