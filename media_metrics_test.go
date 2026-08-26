package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestReportPlaybackKeepsWriteIdentityAcrossRateLimit(t *testing.T) {
	var keys []string
	var points []Metric
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/metrics/report" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		keys = append(keys, r.Header.Get("Idempotency-Key"))
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		var point Metric
		if err := json.NewDecoder(r.Body).Decode(&point); err != nil {
			t.Fatal(err)
		}
		points = append(points, point)
		_, _ = w.Write([]byte(`{"ok":true,"data":{},"error":null,"metadata":{}}`))
	}))
	defer server.Close()
	client := &MetricsClient{HTTP: server.Client(), APIKey: "test-key", Sleep: func(time.Duration) {}, MaxRetries: 2, URL: server.URL + "/v1/metrics/report"}
	err := ReportPlayback(context.Background(), client, PlaybackSnapshot{RunID: "run-7", Region: "apac", Started: 3, PlaybackSeconds: 11})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(keys, []string{"playback:run-7:started", "playback:run-7:started", "playback:run-7:seconds"}) {
		t.Fatalf("keys = %#v", keys)
	}
	if len(points) != 2 || points[0].Type != "counter" || points[1].Type != "gauge" {
		t.Fatalf("points = %#v", points)
	}
}
