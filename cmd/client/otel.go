package main

import (
	"flag"
	"log"

	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

var (
	addr     = flag.String("addr", "localhost:50051", "the server address to connect to")
	promAddr = flag.String("prom-addr", ":9465", "the address of Prometheus server")
)

func MAIN() {
	exporter, err := prometheus.New()
	if err != nil {
		log.Fatalf("failed to initialize prometheus exporter %v", err)
	}

	// Configure MeterProvider for metrics
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	// Configure TracerProvider for tracing (omitted for brevity)
	tp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatalf("failed to initialize stdout trace exporter %v", err)
	}
}
