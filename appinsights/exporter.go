package appinsights

import (
	"context"
	"errors"
	"log"

	"go.opentelemetry.io/otel/sdk/trace"
)

// Exporter is an implementation of an OTel SpanSyncer that uploads spans to Azure Monitor
type Exporter struct {
	process            Process
	tracerOpts         []trace.TracerProviderOption
	instrumentationKey string
	ingestionEndpoint  string
	storage            []*Envelope
}

// ExportSpans exports span data to Azure Monitor
func (e *Exporter) ExportSpans(ctx context.Context, spans []*trace.SpanSnapshot) error {
	envelopes := make([]*Envelope, len(spans))
	for i, span := range spans {
		envelopes[i] = newEnvelopeFromSpan(span, &e.process)
		envelopes[i].IKey = e.instrumentationKey
	}
	res, err := transmit(ctx, nil, e.ingestionEndpoint, envelopes)
	if errors.Is(err, errTransmitRetryable) {
		log.Printf("Transmit retryable: %v, %v", res, err)
		for _, errored := range res.Errors {
			log.Printf("appinsights: failed to transmit item %d: %s", errored.Index, errored.Message)
			e.storage = append(e.storage, envelopes[errored.Index])
		}
	} else if err != nil {
		log.Printf("%v", err)
	}
	if err == nil {
		// transmit from storage
		err = transmitFromStorage(ctx, nil)
	}
	return err
}

// Shutdown stops the exporter flushing any pending exports.
func (e *Exporter) Shutdown(ctx context.Context) error {
	// TODO: transmit from stage for retryable items
	return nil
}
