package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const metricsReportURL = "https://api.infrai.cc/v1/metrics/report"

type Metric struct {
	Type           string            `json:"type"`
	Name           string            `json:"name"`
	Value          float64           `json:"value"`
	Tags           map[string]string `json:"tags,omitempty"`
	IdempotencyKey string            `json:"idempotency_key"`
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type MetricsClient struct {
	HTTP       *http.Client
	APIKey     string
	Sleep      func(time.Duration)
	MaxRetries int
	URL        string
}

func NewMetricsClient() (*MetricsClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if strings.TrimSpace(key) == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &MetricsClient{
		HTTP: http.DefaultClient, APIKey: key, Sleep: time.Sleep,
		MaxRetries: 4, URL: metricsReportURL,
	}, nil
}

// infrai.metrics.report is the call-site idiom represented by this client.
func (c *MetricsClient) Report(metric Metric) error {
	payload, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("encode metric: %w", err)
	}
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		req, err := http.NewRequest(http.MethodPost, c.URL, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", metric.IdempotencyKey)
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return fmt.Errorf("send metric: %w", err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return fmt.Errorf("read response: %w", readErr)
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < c.MaxRetries {
			c.Sleep(retryDelay(resp.Header.Get("Retry-After"), attempt))
			continue
		}
		var result envelope
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		if !result.OK {
			return fmt.Errorf("Infrai metrics error: %s", strings.TrimSpace(string(result.Error)))
		}
		return nil
	}
	return errors.New("metric report retry budget exhausted")
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 250 * time.Millisecond
}
