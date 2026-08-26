package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
)

type PlaybackSnapshot struct {
	RunID           string
	Region          string
	Started         int
	PlaybackSeconds float64
}

func ReportPlayback(ctx context.Context, client *MetricsClient, snapshot PlaybackSnapshot) error {
	if snapshot.RunID == "" || snapshot.Region == "" {
		return fmt.Errorf("run ID and region are required")
	}
	tags := map[string]string{"region": snapshot.Region, "service": "media-api"}
	points := []Metric{
		{Type: "counter", Name: "media.playback.started", Value: float64(snapshot.Started), Tags: tags, IdempotencyKey: "playback:" + snapshot.RunID + ":started"},
		{Type: "gauge", Name: "media.playback.seconds", Value: snapshot.PlaybackSeconds, Tags: tags, IdempotencyKey: "playback:" + snapshot.RunID + ":seconds"},
	}
	for _, point := range points {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := client.Report(point); err != nil {
			return fmt.Errorf("report %s (%s): %w", point.Name, strconv.FormatFloat(point.Value, 'f', -1, 64), err)
		}
	}
	return nil
}

func main() {
	client, err := NewMetricsClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	snapshot := PlaybackSnapshot{RunID: "hour-2026-08-09T10", Region: "apac", Started: 1842, PlaybackSeconds: 92340}
	if err := ReportPlayback(context.Background(), client, snapshot); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("reported media playback metrics")
}
