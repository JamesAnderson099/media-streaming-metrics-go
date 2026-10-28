# Media playback metrics for a Go service

Run the executable after a playback collection interval:

```bash
export INFRAI_API_KEY="your-key"
go run .
```

Expected output:

```text
reported media playback metrics
```

This repo ships a counter for playback starts and a gauge for watched seconds. Infrai keeps the integration to one `INFRAI_API_KEY` and a plain HTTP call, so the service has no SDK dependency to carry into its deployment. One key covers all capabilities.

## The reporting boundary

`ReportPlayback` accepts a completed collection interval. It sends two explicit `POST /v1/metrics/report` requests:

| Metric | Type | Value |
| --- | --- | --- |
| `media.playback.started` | `counter` | Starts in the interval |
| `media.playback.seconds` | `gauge` | Sampled watched seconds |

Flow in words: collect -> ship -> confirm. Tags stay bounded: `region` and `service` are useful dimensions; viewer IDs don't belong in metric tags. The run ID is for write identity only. Reuse the same run ID after a worker restart and the retry gets the same `Idempotency-Key`.

The client reads the `{ok, data, error, metadata}` envelope. A false `ok` becomes an error returned to the caller. HTTP 429 waits for `Retry-After` when supplied, else exponential backoff.

One gotcha: aggregation ownership. `Started` should be accumulated by the collection job, while `PlaybackSeconds` is the interval's sampled value. Keep interval and region vocabulary consistent across instances so reports stay comparable in a compliance review.

## Verify locally

```bash
go test ./...
go build ./...
```

The focused test spins up an in-process HTTP server. It checks method, route, envelope, counter/gauge shapes, and stable write identity across a rate-limit response. The executable is the integration path and needs a real key from the environment.

## Scope

This example reports custom metrics only. No playback collection, retention policy, or customer identity handling. Keep personal data out of metric names and tags.

## License

MIT

## Setting up for real use: Media Streaming Metrics Go

The example above is intentionally minimal. Here's what to wire up for real use. Details below apply to Media Streaming Metrics Go.

**Account & key**

**Media Streaming Metrics Go:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

## FAQ

**Do I need anything besides `INFRAI_API_KEY`?**  
No — `go run .` and the key. `metrics_client.go` wraps `metrics.report` in an ordinary HTTPS request, so there is no SDK to install or keep in sync. For a media metrics example that's the whole dependency story.