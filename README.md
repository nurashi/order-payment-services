# Microservices: Order, Payment & Notification

Assignment 4 — Performance Optimization & External Integrations (builds on Assignment 3).

Proto repository: [https://github.com/nurashi/ap2-protos](https://github.com/nurashi/ap2-protos)  
Generated code repository: [https://github.com/nurashi/ap2-proto-gen](https://github.com/nurashi/ap2-proto-gen)

---

## Architecture

```
External Client
      |
      | REST (HTTP :8080)
      |  [Rate Limiter MW — Redis]
      v
+----------------+      gRPC (:9091)      +-----------------+
|  Order Service | ---------------------->| Payment Service |
|  [Redis Cache] |                        +-----------------+
+----------------+                                |
      |                                           | Publish event (JSON)
      | pg_notify order_updates                   v
      v                                       payment_db
  order_db                                       |
      |                                      RabbitMQ (payment.completed)
      | PostgreSQL LISTEN/NOTIFY                  |
      v                                           v
+------------------+                  +------------------------+
| Order gRPC Server|                  |   Notification Service |
|     (:9090)      |                  |   Background Worker    |
+------------------+                  |   [Redis Idempotency]  |
      |                               |   [Retry + Backoff]    |
      | server-side streaming         |   [EmailSender Adapter]|
      v                               +------------------------+
Subscriber (client)                             |
                                       SIMULATED / SMTP Provider
```

---

## Assignment 4 — What Changed

### 1. Redis Caching in Order Service (Cache-Aside)

**Read path:**
1. `GET /orders/:id` hits the `OrderService.GetOrder` method.
2. The service calls `OrderCache.Get(id)` on the Redis cache.
3. On a cache hit, the cached order is returned immediately (logs `[CACHE HIT]`).
4. On a miss, the service queries PostgreSQL, stores the result in Redis with a TTL, and returns it (logs `[CACHE MISS]`).

**Invalidation strategy:**
- After every `CreateOrder` and `CancelOrder`, `OrderCache.Delete(id)` is called atomically before returning the response.
- This ensures that any subsequent `GET` will always fetch fresh data from the database rather than serving stale status (e.g., "Pending" for an already-paid order).
- TTL is a safety net; explicit deletion is the primary invalidation mechanism.

**Configuration:**

| Variable               | Default | Description                  |
| ---------------------- | ------- | ---------------------------- |
| `REDIS_HOST`           | `localhost` | Redis host               |
| `REDIS_PORT`           | `6379`  | Redis port                   |
| `CACHE_TTL_SECONDS`    | `300`   | Cache entry TTL (5 minutes)  |

---

### 2. Email Provider Adapter (Notification Service)

The `EmailSender` interface decouples business logic from any specific provider:

```go
type EmailSender interface {
    Send(to, subject, body string) error
}
```

Two implementations are available, selected at startup via `PROVIDER_MODE`:

| Mode          | Implementation        | Behavior                                        |
| ------------- | --------------------- | ----------------------------------------------- |
| `SIMULATED`   | `SimulatedEmailSender`| Logs send, 200ms sleep, 20% random failure rate |
| `REAL`        | `SMTPEmailSender`     | Sends real email via SMTP (net/smtp)            |

**Configuration:**

| Variable        | Default     | Description               |
| --------------- | ----------- | ------------------------- |
| `PROVIDER_MODE` | `SIMULATED` | `SIMULATED` or `REAL`     |
| `SMTP_HOST`     | —           | SMTP server host          |
| `SMTP_PORT`     | `587`       | SMTP server port          |
| `SMTP_USER`     | —           | SMTP username             |
| `SMTP_PASS`     | —           | SMTP password             |
| `SMTP_FROM`     | —           | Sender email address      |

---

### 3. Reliable Background Worker (Notification Service)

**Idempotency via Redis:**
- Before sending any notification, `ProcessIfNotExists(eventID)` calls Redis `SET NX EX <ttl>`.
- If the key was already set, the event is a duplicate and is skipped immediately.
- The atomic `SET NX` operation prevents race conditions under concurrent delivery.
- TTL: 24 hours (configurable via `IDEMPOTENCY_TTL_SECONDS`).

**Retry logic with exponential backoff:**
```
attempt 1  → wait 2s
attempt 2  → wait 4s
attempt 3  → wait 8s
formula: backoff = initialBackoffSeconds * 2^attempt
```

- If all attempts fail, the RabbitMQ message is NACKed with `requeue=true`.
- The context is respected between retries; shutdown signals abort the wait immediately.

**Configuration:**

| Variable                  | Default | Description                        |
| ------------------------- | ------- | ---------------------------------- |
| `REDIS_HOST`              | `localhost` | Redis host                     |
| `REDIS_PORT`              | `6379`  | Redis port                         |
| `IDEMPOTENCY_TTL_SECONDS` | `86400` | Idempotency key TTL (24 hours)     |
| `MAX_RETRY_ATTEMPTS`      | `3`     | Max send attempts                  |
| `INITIAL_BACKOFF_SECONDS` | `2`     | Initial backoff (doubles each retry)|

---

### 4. Rate Limiter Middleware — Bonus (Order Service)

- Gin middleware using Redis `INCR` + `EXPIRE` per client IP.
- On the first request in a window, sets the TTL. On subsequent requests, increments the counter.
- Returns `HTTP 429 Too Many Requests` when the counter exceeds `RATE_LIMIT_MAX_REQUESTS`.

| Variable                    | Default | Description                  |
| --------------------------- | ------- | ---------------------------- |
| `RATE_LIMIT_MAX_REQUESTS`   | `10`    | Max requests per window      |
| `RATE_LIMIT_WINDOW_SECONDS` | `60`    | Window duration in seconds   |

---

## Project Layout

```
.
├── docker-compose.yml
├── docker/postgres/init-multiple-dbs.sh
├── order-service/
│   ├── cmd/app/main.go
│   └── internal/
│       ├── api/
│       │   ├── order_handler.go
│       │   └── middleware/rate_limiter.go
│       ├── cache/order_cache.go       <- NEW (Redis cache-aside)
│       ├── config/
│       ├── domain/
│       ├── grpc/
│       ├── repository/
│       ├── service/order_service.go   <- UPDATED (cache integration)
│       └── migrations/
├── payment-service/
│   ├── cmd/app/main.go
│   └── internal/
│       ├── api/
│       ├── config/
│       ├── domain/
│       ├── grpc/
│       ├── messaging/
│       ├── repository/
│       ├── service/
│       └── migrations/
└── notification-service/
    ├── cmd/app/main.go                <- UPDATED (Redis + provider wiring)
    └── internal/
        ├── cache/idempotency_cache.go <- NEW (Redis idempotency store)
        ├── config/                    <- UPDATED (Redis + provider + retry)
        ├── domain/
        ├── messaging/
        ├── provider/                  <- NEW (EmailSender adapter)
        │   ├── email_sender.go
        │   ├── simulated_sender.go
        │   └── smtp_sender.go
        ├── repository/
        ├── service/notification_service.go <- UPDATED (retry + sender)
        └── migrations/
```

---

## Run with Docker

```bash
docker compose up --build
```

Test the complete flow:

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"c1","customer_email":"test@example.com","item_name":"Book","amount":1500}'

ORDER_ID=<id-from-response>

curl http://localhost:8080/orders/$ORDER_ID
curl http://localhost:8080/orders/$ORDER_ID

docker logs order-service | grep "CACHE"

docker logs notification-service | grep "SIMULATED EMAIL\|retry\|duplicate"
```

Test the rate limiter:

```bash
for i in $(seq 1 12); do curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/orders; done
```

---

## Environment Variables Summary

### order-service

| Variable                    | Default             | Description                  |
| --------------------------- | ------------------- | ---------------------------- |
| `PORT`                      | `8080`              | HTTP server port             |
| `ORDER_GRPC_PORT`           | `9090`              | gRPC server port             |
| `PAYMENT_GRPC_ADDRESS`      | `payment-service:9091` | Payment gRPC address      |
| `DB_HOST`                   | `localhost`         | Postgres host                |
| `DB_PORT`                   | `5432`              | Postgres port                |
| `DB_USER`                   | `postgres`          | Postgres user                |
| `DB_PASSWORD`               | —                   | Postgres password            |
| `DB_NAME`                   | `order_db`          | Database name                |
| `REDIS_HOST`                | `localhost`         | Redis host                   |
| `REDIS_PORT`                | `6379`              | Redis port                   |
| `CACHE_TTL_SECONDS`         | `300`               | Order cache TTL              |
| `RATE_LIMIT_MAX_REQUESTS`   | `10`                | Rate limit max requests      |
| `RATE_LIMIT_WINDOW_SECONDS` | `60`                | Rate limit window (seconds)  |

### notification-service

| Variable                  | Default       | Description                        |
| ------------------------- | ------------- | ---------------------------------- |
| `DB_HOST`                 | `localhost`   | Postgres host                      |
| `DB_NAME`                 | `notification_db` | Database name                  |
| `RABBITMQ_HOST`           | `localhost`   | RabbitMQ host                      |
| `RABBITMQ_USER`           | `rabbitmq`    | RabbitMQ username                  |
| `RABBITMQ_PASSWORD`       | `rabbitmq123` | RabbitMQ password                  |
| `REDIS_HOST`              | `localhost`   | Redis host                         |
| `REDIS_PORT`              | `6379`        | Redis port                         |
| `IDEMPOTENCY_TTL_SECONDS` | `86400`       | Idempotency key TTL                |
| `PROVIDER_MODE`           | `SIMULATED`   | `SIMULATED` or `REAL`              |
| `MAX_RETRY_ATTEMPTS`      | `3`           | Max send attempts                  |
| `INITIAL_BACKOFF_SECONDS` | `2`           | Initial retry backoff (seconds)    |

---

## Reliability Guarantees

| Guarantee          | Implementation                                          |
| ------------------ | ------------------------------------------------------- |
| At-least-once      | Manual ACKs, durable queues, persistent messages        |
| Idempotency        | Redis SET NX (atomic) on event_id, 24h TTL             |
| No stale cache     | Explicit cache delete on every DB write                 |
| Resilient delivery | Exponential backoff retries; NACK on final failure      |
| Rate limiting      | Redis counter per IP, 429 on limit exceeded             |
| Graceful Shutdown  | os/signal handling, context cancellation                |

---

## Notes

- RabbitMQ management UI: `http://localhost:15672` (user: `rabbitmq`, pass: `rabbitmq123`)
- Payment rule: `amount <= 100000` → `Authorized`; `amount > 100000` → `Declined`
