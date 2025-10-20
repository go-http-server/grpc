package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/go-http-server/grpc/client"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	oteltracing "google.golang.org/grpc/experimental/opentelemetry"
	otelgrpc "google.golang.org/grpc/stats/opentelemetry"
)

var (
	addr     = flag.String("addr", "localhost:8080", "the server address to connect to")
	promAddr = flag.String("prom-addr", ":9465", "the address of Prometheus server")
)

func MAIN() {
	flag.Parse()
	exporter, err := prometheus.New()
	if err != nil {
		log.Fatalf("failed to initialize prometheus exporter %v", err)
	}

	// r: resource setup
	r, err := resource.New(context.Background(),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithContainer(),
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithAttributes(semconv.ServiceNameKey.String("grpc-client")),
	)
	if err != nil {
		log.Fatalf("failed to create resource: %v", err)
	}

	// Configure MeterProvider for metrics
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
		sdkmetric.WithResource(r),
	)
	// Configure TraceExporter for tracing
	tx, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatalf("failed to initialize stdout trace exporter %v", err)
	}

	// tp: TraceProvider setup
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(tx),
		sdktrace.WithResource(r),
	)

	// Configure W3C Trace Context Propagator for traces
	textMapPropagator := propagation.TraceContext{}
	do := otelgrpc.DialOption(otelgrpc.Options{
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

	// start promhttp handler, serve scrapes on prometheus server
	go http.ListenAndServe(*promAddr, promhttp.Handler())

	cc, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("grpc.NewClient() failed: %v", err)
	}
	defer cc.Close()

	authc := client.NewAuthClient(cc, "admin_valid", "password")
	interceptor, err := client.NewAuthInterceptor(authc, authMethods(), 5*time.Second)
	if err != nil {
		log.Fatalf("Failed to create auth interceptor: %v", err)
	}

	connAuth, err := grpc.NewClient(*addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptor.Unary()),
		grpc.WithStreamInterceptor(interceptor.Stream()),
		do,
	)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer connAuth.Close()

	laptopClient := client.NewLaptopClient(connAuth)
	for {
		testCreateLaptop(laptopClient)
		time.Sleep(5 * time.Second)
	}
}

// func main() {
// 	MAIN()
// }
