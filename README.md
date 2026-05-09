# Notification Management API

A scalable multi-channel notification system built with Go. Handles SMS, Email, and Push notifications with priority queuing, scheduling, message templates, real-time WebSocket status updates, retry logic, and distributed tracing.

## Architecture

```
┌────────────────────────────────────────────────────────────────────┐
│                          HTTP Layer (Gin)                          │
│  /api/v1/notifications  /api/v1/notifications/batch                │
│  /api/v1/templates      /ws  /health  /metrics  /swagger           │
└──────────────────────────────┬─────────────────────────────────────┘
                               │
┌──────────────────────────────▼─────────────────────────────────────┐
│                         Service Layer                              │
│  Recipient validation → Template render → Content validation       │
│  Idempotency check → DB write → Enqueue                           │
└────────────┬──────────────────────────────────────────────────────-┘
             │                                    │
┌────────────▼────────────┐          ┌────────────▼──────────────────┐
│      PostgreSQL          │          │           Redis               │
│  notifications           │          │  notifications:high           │
│  templates               │          │  notifications:normal         │
│  notifications_archive   │          │  notifications:low            │
│  (6 migrations)          │          │  notifications:retry (ZSet)   │
└─────────────────────────┘          └────────────┬──────────────────┘
                                                  │
                                     ┌────────────▼──────────────────┐
                                     │        Worker Pool            │
                                     │  N concurrent goroutines      │
                                     │  Per-channel rate limiter     │
                                     │  Exponential backoff retry    │
                                     └────────────┬──────────────────┘
                                                  │           │
                                     ┌────────────▼────┐  ┌───▼──────────┐
                                     │ External Provider│  │  WS Hub      │
                                     │  (webhook.site)  │  │  broadcasts  │
                                     └─────────────────┘  │  status to   │
                                                          │  subscribers  │
                                                          └──────────────┘
```

**Tech stack:** Go 1.25 · Gin · uber/fx · PostgreSQL · Redis · gorilla/websocket · golang-migrate · OTel (OTLP HTTP → Jaeger) · swaggo

**Key design decisions:**
- `uber/fx` for dependency injection — each layer exposes its own `fx.Module`, `main.go` stays minimal
- Redis List per priority (`notifications:high/normal/low`) with `BRPop` — O(1) ordered consumption
- Redis Sorted Set for durable retry scheduling — score = Unix timestamp, polled every second
- W3C TraceContext injected into the Redis message envelope — traces span the HTTP→worker boundary
- `golang-migrate` with `embed.FS` — migrations baked into the binary, no separate migration container
- Archive worker: completed notifications older than 30 days moved to `notifications_archive` via CTE
- Template system: `{{variable}}` placeholder syntax, resolved at service layer before content validation
- WebSocket hub: fan-out broadcast to all connected clients on every delivery/failure event

## Quick Start

```bash
cp .env.example .env
# Set WEBHOOK_URL to your webhook.site URL
docker compose up --build
```

The API is available at `http://localhost:8080`. Migrations run automatically on startup.

## Running Tests

```bash
# Unit tests
go test ./internal/api/handler/... ./internal/service/... ./internal/domain ./internal/metrics/... ./internal/ws/...

# All tests including integration (requires Docker)
make test-integration

# With coverage report
go test -coverprofile=coverage.out ./internal/... && go tool cover -html=coverage.out
```

Unit test coverage: **97.9%**

CI runs automatically on every push and pull request (see `.github/workflows/ci.yml`). The pipeline fails if coverage drops below 95%.

## API Reference

Swagger UI: `http://localhost:8080/swagger/index.html`

### Notifications

#### Create a notification

```bash
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "recipient": "+905551234567",
    "channel": "sms",
    "content": "Your OTP is 123456",
    "priority": "high",
    "idempotency_key": "order-42-sms"
  }'
```

Or use a template instead of inline content:

```bash
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "recipient": "+905551234567",
    "channel": "sms",
    "template_id": "550e8400-e29b-41d4-a716-446655440000",
    "variables": { "name": "Kenan", "code": "7890" }
  }'
```

```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "recipient": "+905551234567",
  "channel": "sms",
  "content": "Hello Kenan, your OTP is 7890",
  "priority": "normal",
  "status": "queued",
  "created_at": "2026-05-09T10:00:00Z",
  "updated_at": "2026-05-09T10:00:00Z"
}
```

**Fields:**

| Field | Required | Description |
|-------|----------|-------------|
| `recipient` | Yes | E.164 phone (`+905551234567`) for SMS/Push, email for Email |
| `channel` | Yes | `sms`, `email`, or `push` |
| `content` | One of | Inline message text |
| `template_id` | One of | UUID of a saved template |
| `variables` | No | Key-value pairs for `{{placeholder}}` substitution |
| `priority` | No | `high`, `normal` (default), or `low` |
| `idempotency_key` | No | Returns existing record if key already used |
| `scheduled_at` | No | RFC3339 timestamp for future delivery |

#### Create a batch (up to 1000)

```bash
curl -X POST http://localhost:8080/api/v1/notifications/batch \
  -H "Content-Type: application/json" \
  -d '{
    "notifications": [
      {"recipient": "+905551234567", "channel": "sms",   "content": "Flash sale!"},
      {"recipient": "user@example.com", "channel": "email", "content": "Flash sale!"}
    ]
  }'
```

```json
{ "batch_id": "...", "total": 2, "queued": 2 }
```

#### Schedule a notification

```bash
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "recipient": "+905551234567",
    "channel": "sms",
    "content": "Your appointment is in 1 hour",
    "scheduled_at": "2026-05-10T09:00:00Z"
  }'
```

Status is set to `pending` until the scheduled time, then the scheduler moves it to `queued`.

#### Other notification endpoints

```bash
# Get by ID
curl http://localhost:8080/api/v1/notifications/{id}

# List with filters and pagination
curl "http://localhost:8080/api/v1/notifications?status=failed&channel=sms&page=1&page_size=20"
curl "http://localhost:8080/api/v1/notifications?batch_id={batch_id}&date_from=2026-05-01T00:00:00Z"

# Cancel (only pending/queued)
curl -X DELETE http://localhost:8080/api/v1/notifications/{id}
```

---

### Templates

Create reusable message templates with `{{variable}}` placeholders. Templates are channel-scoped — an SMS template cannot be used for an email notification.

#### Create a template

```bash
curl -X POST http://localhost:8080/api/v1/templates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "otp_sms",
    "channel": "sms",
    "body": "Hello {{name}}, your verification code is {{code}}. Valid for 5 minutes."
  }'
```

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "otp_sms",
  "channel": "sms",
  "body": "Hello {{name}}, your verification code is {{code}}. Valid for 5 minutes.",
  "created_at": "2026-05-09T10:00:00Z",
  "updated_at": "2026-05-09T10:00:00Z"
}
```

#### Other template endpoints

```bash
# List all templates
curl http://localhost:8080/api/v1/templates

# Get by ID
curl http://localhost:8080/api/v1/templates/{id}

# Delete
curl -X DELETE http://localhost:8080/api/v1/templates/{id}
```

---

### WebSocket — Real-time Status Updates

Connect to `/ws` to receive live delivery events for all notifications.

```js
const ws = new WebSocket("ws://localhost:8080/ws");

ws.onmessage = (event) => {
  const update = JSON.parse(event.data);
  console.log(update);
  // {
  //   "notification_id": "660e8400-...",
  //   "status": "delivered",
  //   "timestamp": "2026-05-09T10:00:01.234Z"
  // }
};
```

Events are broadcast to all connected clients when a notification transitions to `delivered` or `failed`. The connection stays open; reconnect on close.

---

### Metrics

```bash
curl http://localhost:8080/metrics
```

```json
{
  "queue_depth": { "high": 0, "normal": 5, "low": 2, "total": 7 },
  "delivery": {
    "delivered_total": 1024,
    "failed_total": 3,
    "success_rate": 99.71,
    "avg_latency_ms": 142.5,
    "by_channel": {
      "sms":   { "delivered": 800, "failed": 2 },
      "email": { "delivered": 200, "failed": 1 },
      "push":  { "delivered": 24,  "failed": 0 }
    }
  }
}
```

### Health

```bash
curl http://localhost:8080/health
# 200 OK — both Postgres and Redis are reachable
# 503 Service Unavailable — one or both dependencies are down
```

---

## Notification Status Flow

```
created → pending (scheduled) → queued → processing → delivered
                                       ↘              ↗
                                        → queued (retry, up to 3x)
                                        → failed (max retries exceeded)
                                        → cancelled (via DELETE)
```

---

## Configuration

All configuration via environment variables (see `.env.example`):

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_DSN` | — | PostgreSQL connection string (required) |
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `WEBHOOK_URL` | — | External provider endpoint (required) |
| `SERVER_PORT` | `8080` | HTTP listen port |
| `WORKER_CONCURRENCY` | `10` | Parallel worker goroutines |
| `WORKER_RATE_LIMIT_PER_SEC` | `100` | Max deliveries per second per channel |
| `WORKER_MAX_RETRIES` | `3` | Max delivery attempts before `failed` |
| `WORKER_RETRY_BASE_DELAY` | `5s` | Exponential backoff base (delay = 2ⁿ × base) |
| `WEBHOOK_TIMEOUT` | `10s` | External provider HTTP timeout |
| `ARCHIVE_AFTER` | `720h` | Age threshold for archiving completed notifications |
| `ARCHIVE_BATCH_SIZE` | `1000` | Max rows moved per archive cycle |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4318` | Jaeger OTLP HTTP endpoint |
| `OTEL_SERVICE_NAME` | `notification-api` | Service name in traces |

## Content Limits

| Channel | Max characters |
|---------|---------------|
| SMS | 160 |
| Push | 256 |
| Email | 10,000 |

Limits are enforced on rune count (Unicode-aware). Exceeding the limit returns `422 Unprocessable Entity`.

## Retry Logic

Failed deliveries are retried with exponential backoff via a Redis Sorted Set (`notifications:retry`). The scheduler polls every second and re-enqueues due messages.

| Attempt | Delay |
|---------|-------|
| 1st retry | 5s |
| 2nd retry | 10s |
| 3rd retry | 20s |
| After 3rd | Status → `failed`, WebSocket event broadcast |

## Database Scaling

At millions of notifications/day the following indexes keep queries fast:

```sql
-- List API: status filter + ORDER BY created_at (avoids sort step)
CREATE INDEX idx_notifications_status_created ON notifications(status, created_at DESC);

-- List API: combined status + channel filter
CREATE INDEX idx_notifications_status_channel ON notifications(status, channel);

-- Scheduled notifications: partial index, only pending rows with a scheduled_at
CREATE INDEX idx_notifications_scheduled ON notifications(scheduled_at)
  WHERE scheduled_at IS NOT NULL AND status = 'pending';
```

Completed records older than 30 days are moved in batches to `notifications_archive` by a background worker, keeping the hot table small.

## Observability

### Distributed Tracing

All requests are traced via OpenTelemetry (OTLP HTTP → Jaeger). Traces span the HTTP→worker boundary: the trace context is injected into the Redis message envelope so worker spans are linked to the originating HTTP request.

After `docker compose up`, open Jaeger UI at `http://localhost:16686`. Select the `notification-api` service and click **Find Traces**.

Every log line includes `trace_id` so you can jump from a log entry directly to its Jaeger trace:

```json
{"level":"info","msg":"notification delivered","id":"...","trace_id":"4bf92f3577b34da6a3ce929d0e0e4736"}
```

### Structured Logging

All logs are JSON (zap). Key fields:

| Field | Present in | Description |
|-------|-----------|-------------|
| `trace_id` | all worker & HTTP logs | Links log to Jaeger trace |
| `request_id` | HTTP request logs | Per-request correlation ID |
| `id` | worker logs | Notification UUID |
| `channel` | worker logs | `sms`, `email`, `push` |
| `latency` | HTTP & delivery logs | Duration |

### Metrics

`GET /metrics` returns in-memory counters (no Prometheus dependency):

```bash
curl http://localhost:8080/metrics
```

