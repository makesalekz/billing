package tracing

import (
	"context"

	u_config "gitlab.calendaria.team/services/utils/v1/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

type Tracer struct {
	initialized bool
	conf        *u_config.Config
}

func NewTracer(conf *u_config.Config) *Tracer {
	return &Tracer{
		conf: conf,
	}
}

// Initialize the OpenTelemetry tracer with the OTLP span exporter.
// The OTLP exporter is selected based on the OTLP_GRPC_ADDRESS or OTLP_HTTP_ADDRESS environment variable.
// If neither environment variable is set, the tracer is not initialized.
func (t *Tracer) Initialize() error {
	// If the tracer is already initialized, return early
	if t.initialized {
		return nil
	}

	// Create new OTLP trace exporter
	var exp *otlptrace.Exporter
	var err error
	endpoint, _ := t.conf.Value("OTLP_GRPC_ADDRESS").String()
	if endpoint == "" {
		endpoint, _ = t.conf.Value("OTLP_HTTP_ADDRESS").String()
		if endpoint == "" {
			return nil
		}
		exp, err = otlptracehttp.New(context.Background(), otlptracehttp.WithEndpoint(endpoint), otlptracehttp.WithInsecure())
		if err != nil {
			return err
		}
	} else {
		exp, err = otlptracegrpc.New(context.Background(), otlptracegrpc.WithEndpoint(endpoint), otlptracegrpc.WithInsecure())
		if err != nil {
			return err
		}
	}

	// Create new OTLP trace provider
	tp := tracesdk.NewTracerProvider(
		// Set the sampling rate based on the parent span to 100%
		tracesdk.WithSampler(tracesdk.ParentBased(tracesdk.TraceIDRatioBased(1.0))),
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exp),
		// Record information about this application in an Resource.
		tracesdk.WithResource(resource.NewSchemaless(
			semconv.ServiceNameKey.String(t.conf.GetAppName()),
			attribute.String("exporter", "jaeger"),
		)),
	)

	// Register the trace provider with the global trace provider
	otel.SetTracerProvider(tp)
	t.initialized = true
	return nil
}

func (t *Tracer) IsInitialized() bool {
	return t.initialized
}
