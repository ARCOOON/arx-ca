package telemetry

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	defaultServiceName   = "arx-ca"
	defaultOTLPEndpoint  = "http://localhost:4318"
	defaultShutdownGrace = 5 * time.Second
)

var (
	meter             metric.Meter
	requestDuration   metric.Float64Histogram
	requestTotal      metric.Int64Counter
	shutdownTelemetry func(context.Context) error
)

// Config holds OpenTelemetry exporter settings resolved from the environment.
type Config struct {
	Enabled     bool
	ServiceName string
	Endpoint    string
	Insecure    bool
}

// LoadConfigFromEnv reads standard OTEL_* variables with arx-ca defaults.
func LoadConfigFromEnv() Config {
	if strings.EqualFold(os.Getenv("OTEL_SDK_DISABLED"), "true") {
		return Config{Enabled: false}
	}

	endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("ARX_CA_OTEL_ENDPOINT"))
	}
	if endpoint == "" {
		endpoint = defaultOTLPEndpoint
	}

	serviceName := strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME"))
	if serviceName == "" {
		serviceName = defaultServiceName
	}

	insecure := strings.EqualFold(os.Getenv("OTEL_EXPORTER_OTLP_INSECURE"), "true")
	if !insecure && strings.HasPrefix(strings.ToLower(endpoint), "http://") {
		insecure = true
	}

	return Config{
		Enabled:     true,
		ServiceName: serviceName,
		Endpoint:    endpoint,
		Insecure:    insecure,
	}
}

// Init configures OTLP trace and metric exporters and registers global providers.
func Init(ctx context.Context) (func(context.Context) error, error) {
	cfg := LoadConfigFromEnv()
	if !cfg.Enabled {
		log.Println("telemetry: OpenTelemetry disabled")
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create otel resource: %w", err)
	}

	traceOpts := []otlptracehttp.Option{
		otlptracehttp.WithEndpointURL(cfg.Endpoint),
	}
	if cfg.Insecure {
		traceOpts = append(traceOpts, otlptracehttp.WithInsecure())
	}

	traceExporter, err := otlptracehttp.New(ctx, traceOpts...)
	if err != nil {
		return nil, fmt.Errorf("create otlp trace exporter: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)

	metricOpts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpointURL(cfg.Endpoint),
	}
	if cfg.Insecure {
		metricOpts = append(metricOpts, otlpmetrichttp.WithInsecure())
	}

	metricExporter, err := otlpmetrichttp.New(ctx, metricOpts...)
	if err != nil {
		_ = tracerProvider.Shutdown(ctx)
		return nil, fmt.Errorf("create otlp metric exporter: %w", err)
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(meterProvider)

	meter = meterProvider.Meter("github.com/your-org/arx-ca/http")
	requestDuration, err = meter.Float64Histogram(
		"http.server.duration",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		_ = tracerProvider.Shutdown(ctx)
		_ = meterProvider.Shutdown(ctx)
		return nil, fmt.Errorf("create request duration histogram: %w", err)
	}

	requestTotal, err = meter.Int64Counter(
		"http.server.request.count",
		metric.WithDescription("Total HTTP requests handled"),
	)
	if err != nil {
		_ = tracerProvider.Shutdown(ctx)
		_ = meterProvider.Shutdown(ctx)
		return nil, fmt.Errorf("create request counter: %w", err)
	}

	shutdown := func(ctx context.Context) error {
		traceErr := tracerProvider.Shutdown(ctx)
		metricErr := meterProvider.Shutdown(ctx)
		if traceErr != nil {
			return traceErr
		}
		return metricErr
	}
	shutdownTelemetry = shutdown

	log.Printf("telemetry: OpenTelemetry enabled service=%s endpoint=%s", cfg.ServiceName, cfg.Endpoint)
	return shutdown, nil
}

// Shutdown flushes telemetry exporters.
func Shutdown(ctx context.Context) error {
	if shutdownTelemetry == nil {
		return nil
	}
	shutdownCtx, cancel := context.WithTimeout(ctx, defaultShutdownGrace)
	defer cancel()
	return shutdownTelemetry(shutdownCtx)
}

// HTTPMiddleware instruments the HTTP handler with OpenTelemetry traces and metrics.
func HTTPMiddleware(next http.Handler) http.Handler {
	if meter == nil {
		return next
	}

	traced := otelhttp.NewHandler(next, "arx-ca",
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + r.URL.Path
		}),
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		traced.ServeHTTP(recorder, r)

		attrs := []attribute.KeyValue{
			attribute.String("http.method", r.Method),
			attribute.String("http.route", r.URL.Path),
			attribute.Int("http.status_code", recorder.status),
		}
		requestDuration.Record(r.Context(), time.Since(start).Seconds(), metric.WithAttributes(attrs...))
		requestTotal.Add(r.Context(), 1, metric.WithAttributes(attrs...))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
