# OpenTelemetry Azure Monitor

## Installation

```sh
go get github.com/adamko147/opentelemetry-azure-monitor
```

## Usage

### Trace

The **Azure Monitor Span Exporter** allows you to export [OpenTelemetry](https://opentelemetry.io/) traces to [Azure Monitor](https://docs.microsoft.com/azure/azure-monitor/).

This example shows how to send a span "hello" to Azure Monitor.

```go
package main

import (
	"context"
	"log"

	appinsights "github.com/adamko147/opentelemetry-azure-monitor/appinsights"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	shutdown, err := appinsights.InstallNewPipeline(
		appinsights.WithProcess(appinsights.Process{
			ServiceName: "trace-demo",
		}),
		appinsights.WithInstrumentationKey('<instrumentation-key>')
		appinsights.WithSDK(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer shutdown()

	ctx := context.Background()
	tracer := otel.Tracer("my-module")
	_, span := tracer.Start(ctx, "operation")
	log.Println("Hello World")
	span.SetStatus(codes.Ok, "Succeeded")
	span.End()
}
```
