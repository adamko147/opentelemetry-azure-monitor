# OpenTelemetry Azure Monitor

## Installation

```sh
go get github.com/adamko147/opentelemetry-azure-monitor
```

## Usage

### Trace

The **Azure Monitor Span Exporter** allows you to export [OpenTelemetry](https://opentelemetry.io/) traces to [Azure Monitor](https://docs.microsoft.com/azure/azure-monitor/).

This example shows how to send a span "hello" to Azure Monitor.

* Create an Azure Monitor resource and get the instrumentation key, more information can be found [here](https://docs.microsoft.com/azure/azure-monitor/app/create-new-resource).
* Place your instrumentation connection string in a `connection string` and directly into your code.
* Alternatively, you can specify your `connection string` in an environment variable `APPLICATIONINSIGHTS_CONNECTION_STRING`.

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
