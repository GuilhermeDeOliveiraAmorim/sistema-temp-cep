package main

import (
	"context"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/trace"
)

// initTracer configura o OpenTelemetry para usar o Zipkin como exportador
func initTracer() func() {
	exporter, err := zipkin.New(
		"http://zipkin:9411/api/v2/spans",
		zipkin.WithLogger(log.Default()),
	)
	if err != nil {
		log.Fatalf("failed to create Zipkin exporter: %v", err)
	}

	provider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithSampler(trace.AlwaysSample()),
	)
	otel.SetTracerProvider(provider)
	return func() { _ = provider.Shutdown(context.Background()) }
}
