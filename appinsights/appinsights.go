package appinsights

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Process contains the information exported to Azure Monitor about the source of the trace data.
type Process struct {
	ServiceName string
	Tags        []attribute.KeyValue
}

// Option is type definion for WithXxx functions
type Option func(*Exporter)

// WithConnectionStringFromEnv sets the connection from environment variable
func WithConnectionStringFromEnv() Option {
	return WithConnectionString(os.Getenv("APPLICATIONINSIGHTS_CONNECTION_STRING"))
}

// WithConnectionString sets the connection string for exporter
func WithConnectionString(cs string) Option {
	return func(e *Exporter) {
		if ep, ikey, err := parseConnectionString(cs); err == nil {
			if ep != "" {
				e.ingestionEndpoint = ep
			}
			if ikey != "" {
				e.instrumentationKey = ikey
			}
		}
	}
}

// WithProcess sets the process with the information about the exporting process.
func WithProcess(process Process) Option {
	return func(e *Exporter) {
		e.process = process
	}
}

// WithInstrumentationKey set the instrumentation key for ingestion endpoint
func WithInstrumentationKey(key string) Option {
	return func(e *Exporter) {
		e.instrumentationKey = key
	}
}

// WithInstrumentationKeyFromEnv sets the instrumentation key from environment variable
func WithInstrumentationKeyFromEnv() Option {
	return WithInstrumentationKey(os.Getenv("APPINSIGHTS_INSTRUMENTATIONKEY"))
}

// WithEndpoint sets the ingestion endpoint for the exporter
func WithEndpoint(url string) Option {
	return func(e *Exporter) {
		e.ingestionEndpoint = url
	}
}

func parseConnectionString(cs string) (endpoint, ikey string, err error) {
	var suffix, location string
	pairs := strings.Split(strings.ToLower(cs), ";")
	for _, pair := range pairs {
		v := strings.Split(pair, "=")
		if len(v) >= 2 {
			switch v[0] {
			case "ingestionendpoint":
				endpoint = strings.Join(v[1:], "=")
			case "instrumentationkey":
				ikey = strings.Join(v[1:], "=")
			case "endpointsuffix":
				suffix = strings.Join(v[1:], "=")
			case "location":
				location = strings.Join(v[1:], "=")
			}
		}
	}
	if endpoint == "" {
		var endpointSuffix, locationPrefix string
		if suffix != "" {
			endpointSuffix = suffix
			if location != "" {
				locationPrefix = location
			}
			endpoint = fmt.Sprintf("https://%sdc.%s", locationPrefix, endpointSuffix)
		} else {
			endpoint = "https://dc.services.visualstudio.com"
		}
	}
	endpoint = fmt.Sprintf("%s/v2/track", strings.TrimRight(endpoint, " /"))
	return
}

// WithOptions sets the TracerProviderOptions for the exporter pipeline.
func WithOptions(c ...sdktrace.TracerProviderOption) Option {
	return func(e *Exporter) {
		e.tracerOpts = append(e.tracerOpts, c...)
	}
}

// NewExporter returns an OTel Exporter implementation that exports the
// collected spans to Azure Monitor.
func NewExporter(opts ...Option) (*Exporter, error) {
	e := &Exporter{}
	for _, opt := range opts {
		opt(e)
	}
	return e, nil
}

// NewExportPipeline sets up a complete export pipeline
// with the recommended setup for trace provider
func NewExportPipeline(opts ...Option) (trace.TracerProvider, func(context.Context) error, error) {
	opts = append([]Option{WithConnectionStringFromEnv()}, opts...)
	exporter, err := NewExporter(opts...)
	if err != nil {
		return nil, nil, err
	}

	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tpo := []sdktrace.TracerProviderOption{sdktrace.WithSpanProcessor(bsp)}
	tpo = append(tpo, exporter.tracerOpts...)
	tp := sdktrace.NewTracerProvider(tpo...)
	return tp, bsp.Shutdown, nil
}

// InstallNewPipeline instantiates a NewExportPipeline with the
// recommended configuration and registers it globally.
func InstallNewPipeline(opts ...Option) (func(context.Context) error, error) {
	tp, shutdown, err := NewExportPipeline(opts...)
	if err != nil {
		return nil, err
	}

	otel.SetTracerProvider(tp)
	return shutdown, nil
}
