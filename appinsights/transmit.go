package appinsights

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var (
	errTransmitRetryable = errors.New("appinsights")
	errTransmitFailed    = errors.New("appinsights")
)

type transmitError struct {
	Index      int    `json:"index"`
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
}

type transmitResult struct {
	Received   int              `json:"itemsReceived"`
	Accepted   int              `json:"itemsAccepted"`
	Errors     []*transmitError `json:"errors,omitempty"`
	retryAfter *time.Time
}

func transmit(ctx context.Context, client *http.Client, endpoint string, envelopes []*Envelope) (*transmitResult, error) {
	if client == nil {
		client = http.DefaultClient
	}
	result := &transmitResult{}

	// skip logging for now, handle using Logger interface later
	// defer func() {
	// 	b, _ := json.Marshal(result)
	// 	log.Println("appinsights", string(b))
	// }()

	buf, err := json.Marshal(envelopes)
	if err != nil {
		return result, fmt.Errorf("%w: failed to marshall envelopes: %v", errTransmitFailed, err)
	}
	body := bytes.NewBuffer(buf)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, body)
	if err != nil {
		return result, fmt.Errorf("%w: failed to create new request: %v", errTransmitFailed, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("%w: failed to send request: %v", errTransmitRetryable, err)
	}
	defer res.Body.Close()
	if r, ok := res.Header[http.CanonicalHeaderKey("Retry-After")]; ok && len(r) == 1 {
		if t, err := time.Parse(time.RFC1123, r[0]); err == nil {
			result.retryAfter = &t
		}
	}
	if res.StatusCode == http.StatusOK {
		result.Received, result.Accepted = len(envelopes), len(envelopes)
		return result, nil
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return result, fmt.Errorf("%w: failed to decode response: %v", errTransmitRetryable, err)
	}
	if res.StatusCode == http.StatusPartialContent ||
		res.StatusCode == http.StatusTooManyRequests ||
		res.StatusCode == http.StatusInternalServerError ||
		res.StatusCode == http.StatusServiceUnavailable {
		return result, fmt.Errorf("%w: StatusCode=%d", errTransmitRetryable, res.StatusCode)
	}
	return result, fmt.Errorf("%w: status=%d", errTransmitFailed, res.StatusCode)
}

func transmitFromStorage(ctx context.Context, client *http.Client) error {
	return nil
}
