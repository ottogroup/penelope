package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/oauth"
)

const serviceName = "penelope"

// NewOTLPTracerProvider creates a vendor-neutral tracer provider that is configured entirely
// through the standard OpenTelemetry environment variables (OTEL_EXPORTER_OTLP_*, OTEL_SERVICE_NAME,
// OTEL_RESOURCE_ATTRIBUTES, OTEL_TRACES_SAMPLER, ...).
func NewOTLPTracerProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
	res, err := resource.New(
		ctx,
		resource.WithTelemetrySDK(),
		resource.WithAttributes(semconv.ServiceNameKey.String(serviceName)),
		// Last, so OTEL_SERVICE_NAME / OTEL_RESOURCE_ATTRIBUTES override the defaults above.
		resource.WithFromEnv(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenTelemetry resource: %w", err)
	}

	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
	), nil
}

// NewGCPTracerProvider creates a tracer provider that exports to Google Cloud Telemetry
// using Application Default Credentials.
func NewGCPTracerProvider(ctx context.Context, projectID string) (*sdktrace.TracerProvider, error) {
	creds, err := oauth.NewApplicationDefault(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load application default credentials: %w", err)
	}

	res, err := resource.New(
		ctx,
		resource.WithDetectors(gcp.NewDetector()),
		resource.WithTelemetrySDK(),
		resource.WithFromEnv(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			attribute.String("gcp.project_id", projectID),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenTelemetry resource: %w", err)
	}

	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint("telemetry.googleapis.com:443"),
		otlptracegrpc.WithDialOption(grpc.WithPerRPCCredentials(creds)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	), nil
}
