package framework_gin

// see https://github.com/SigNoz/sample-golang-app

import (
	"context"
	"strings"

	"github.com/abstratium-informatique-sarl/stratis/pkg/env"
	"github.com/abstratium-informatique-sarl/stratis/pkg/logging"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const (
	collectorURL = "maxant-jaegerallinone:4317"
	insecure     = "true"
)

// custom metrics? see here: https://github.com/SigNoz/sample-golang-app/blob/master/metrics/metrics.go

// custom span (from https://github.com/go-gorm/opentelemetry/blob/master/examples/demo/main.go):
// (or indeed https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/instrumentation/github.com/gin-gonic/gin/otelgin/example/server.go)
// or https://opentelemetry.io/docs/languages/go/instrumentation/

func InitTracer(prefix string, router *gin.Engine) func(context.Context) error {
	ServiceName := prefix + "-" + env.Getenv()
	log := logging.GetLog("framework_gin");

	var secureOption otlptracegrpc.Option

	if strings.ToLower(insecure) == "false" || insecure == "0" || strings.ToLower(insecure) == "f" {
		secureOption = otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, ""))
	} else {
		secureOption = otlptracegrpc.WithInsecure()
	}

	exporter, err := otlptrace.New(
		context.Background(),
		otlptracegrpc.NewClient(
			secureOption,
			otlptracegrpc.WithEndpoint(collectorURL),
		),
	)

	if err != nil {
		log.Fatal().Msgf("Failed to create exporter: %v", err)
	}
	resources, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNamespaceKey.String(ServiceName), // jaeger, https://stackoverflow.com/a/77755998/458370
			attribute.String("service.name", ServiceName), // grafana tempo?
			attribute.String("library.language", "go"),
		),
	)
	if err != nil {
		log.Fatal().Msgf("Could not set resources: %v", err)
	}

	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(resources),
		),
	)
    router.Use(otelgin.Middleware(ServiceName))
	return exporter.Shutdown
}
