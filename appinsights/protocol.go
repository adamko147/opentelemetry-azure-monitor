package appinsights

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel/codes"
	export "go.opentelemetry.io/otel/sdk/export/trace"
	"go.opentelemetry.io/otel/semconv"
	trace "go.opentelemetry.io/otel/trace"
)

var (
	hostname, _ = os.Hostname()
	platform    = "darwin"
)

const (
	remoteDependency = "Microsoft.Applicationappinsights.RemoteDependency"
)

// Envelope represents system variables for a telemetry item.
type Envelope struct {
	Ver        int               `json:"ver"`
	Name       string            `json:"name"`
	Time       string            `json:"time"`
	SampleRate float64           `json:"sampleRate,omitempty"`
	Seq        string            `json:"seq,omitempty"`
	IKey       string            `json:"iKey"`
	Tags       map[string]string `json:"tags,omitempty"`
	Data       *Data             `json:"data"`
}

// RemoteDependencyData represents internal/client span
type RemoteDependencyData struct {
	Ver          int                 `json:"ver"`
	Name         string              `json:"name"`
	ID           string              `json:"id"`
	ResultCode   string              `json:"resultCode"`
	Duration     string              `json:"duration"`
	Success      bool                `json:"success"`
	Data         string              `json:"data"`
	Target       string              `json:"target"`
	Type         string              `json:"type"`
	Properties   *map[string]string  `json:"properties"`
	Measurements *map[string]float64 `json:"measurements"`
}

// RequestData represents consumer/server span
type RequestData struct {
	Ver          int                 `json:"ver"`
	ID           string              `json:"id"`
	Source       string              `json:"source"`
	Name         string              `json:"name"`
	Duration     string              `json:"duration"`
	ResponseCode string              `json:"responseCode"`
	Success      bool                `json:"success"`
	URL          string              `json:"url"`
	Properties   *map[string]string  `json:"properties,omitempty"`
	Measurements *map[string]float64 `json:"measurements,omitempty"`
}

// Data represesent span data in transmittion
type Data struct {
	BaseType string      `json:"baseType"`
	BaseData interface{} `json:"baseData"`
}

func fmtDuration(duration time.Duration) string {
	duration = duration.Round(time.Nanosecond)
	h := duration / time.Hour
	duration -= h * time.Hour
	d, h := h/24, h%24
	m := duration / time.Minute
	duration -= m * time.Minute
	s := duration / time.Second
	duration -= s * time.Second
	us := duration / time.Microsecond
	return fmt.Sprintf("%d.%02d:%02d:%02d.%06d", d, h, m, s, us)
}

// NewEnvelope creates new envelope
func newEnvelopeFromSpan(span *export.SpanSnapshot, process *Process) *Envelope {
	envelope := &Envelope{
		Ver: 1,
		Tags: map[string]string{
			"ai.cloud.role":         process.ServiceName,
			"ai.cloud.roleInstance": hostname,
			"ai.device.id":          hostname,
			"ai.device.osVersion":   platform,
		},
		Time: span.StartTime.UTC().Format("2006-01-02T15:04:05.000000Z"),
	}
	envelope.Tags["ai.operation.id"] = span.SpanContext.TraceID.String()
	if span.ParentSpanID.IsValid() {
		envelope.Tags["ai.operation.parentId"] = span.ParentSpanID.String()
	}
	props := make(map[string]string, len(span.Attributes))
	for _, a := range span.Attributes {
		props[string(a.Key)] = a.Value.AsString()
	}

	if span.SpanKind == trace.SpanKindConsumer || span.SpanKind == trace.SpanKindServer {
		envelope.Name = "Microsoft.Applicationappinsights.Request"
		data := &RequestData{
			Ver:          2,
			ID:           span.SpanContext.SpanID.String(),
			Duration:     fmtDuration(span.EndTime.Sub(span.StartTime)),
			ResponseCode: fmt.Sprintf("%d", span.StatusCode),
			Success:      span.StatusCode == codes.Ok,
		}
		var method, route, url, host, path, scheme string
		status := -1
		for _, attr := range span.Attributes {
			switch attr.Key {
			case semconv.HTTPMethodKey:
				method = attr.Value.AsString()
			case semconv.HTTPRouteKey:
				route = attr.Value.AsString()
			case semconv.HTTPTargetKey:
				path = attr.Value.AsString()
			case semconv.HTTPHostKey:
				host = attr.Value.AsString()
			case semconv.HTTPSchemeKey:
				scheme = attr.Value.AsString()
			case semconv.HTTPURLKey:
				url = attr.Value.AsString()
			case semconv.HTTPStatusCodeKey:
				status = int(attr.Value.AsInt32())
			}
		}
		if method != "" {
			data.Name = method
			if route != "" {
				data.Name = fmt.Sprintf("%s %s", data.Name, route)
				envelope.Tags["ai.operation.name"] = data.Name
			} else if path != "" {
				data.Name = fmt.Sprintf("%s %s", data.Name, route)
				envelope.Tags["ai.operation.name"] = data.Name
			}
			props["request.name"] = data.Name
		}
		if url == "" && scheme != "" && host != "" && path != "" {
			url = fmt.Sprintf("%s://%s/%s", scheme, host, strings.TrimLeft(path, "/"))
		}
		if url != "" {
			data.URL = url
			props["request.url"] = url
		}
		if status != -1 {
			data.ResponseCode = fmt.Sprintf("%d", status)
			data.Success = 200 <= status && status < 400
		}
		if len(props) > 0 {
			data.Properties = &props
		}
		envelope.Data = &Data{
			BaseType: "RequestData",
			BaseData: data,
		}
	} else {
		envelope.Name = remoteDependency
		data := &RemoteDependencyData{
			Ver:        2,
			Name:       span.Name,
			ID:         span.SpanContext.SpanID.String(),
			ResultCode: fmt.Sprintf("%d", span.StatusCode),
			Duration:   fmtDuration(span.EndTime.Sub(span.StartTime)),
			Type:       "InProc",
			Success:    span.StatusCode == codes.Ok,
		}
		if span.SpanKind == trace.SpanKindClient || span.SpanKind == trace.SpanKindProducer {
			var method string
			var url *url.URL
			status := -1
			for _, attr := range span.Attributes {
				switch attr.Key {
				case semconv.HTTPMethodKey:
					method = attr.Value.AsString()
				case semconv.HTTPURLKey:
					url, _ = url.Parse(attr.Value.AsString())
				case semconv.HTTPStatusCodeKey:
					status = int(attr.Value.AsInt32())
				}
			}
			if url != nil {
				data.Type = "HTTP"
				data.Data = url.String()
				data.Target = url.Host
				if method != "" {
					data.Name = fmt.Sprintf("%s %s://%s%s", method, url.Scheme, url.Host, url.Path)
				}
				if status != -1 {
					data.ResultCode = fmt.Sprintf("%d", status)
					data.Success = 200 <= status && status < 400
				}
			}
		}
		if len(props) > 0 {
			data.Properties = &props
		}
		envelope.Data = &Data{
			BaseType: "RemoteDependencyData",
			BaseData: data,
		}
	}
	return envelope
}
