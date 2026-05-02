# Microservices: Order & Payment with Event-Driven Notifications

Assignment 3 — Event-Driven Architecture with Message Queues.

Proto repository: [https://github.com/nurashi/ap2-protos](https://github.com/nurashi/ap2-protos)  
Generated code repository: [https://github.com/nurashi/ap2-proto-gen](https://github.com/nurashi/ap2-proto-gen)

---

## Architecture

```
External Client
      |
      | REST (HTTP :8080)
      v
+----------------+      gRPC (:9091)      +-----------------+
|  Order Service | ---------------------->| Payment Service |
+----------------+                        +-----------------+
      |                                         |
      | pg_notify order_updates                 | Publish event (JSON)
      v                                         v
  order_db                                  payment_db
      |                                         |
      | PostgreSQL LISTEN/NOTIFY                | RabbitMQ (payment.completed)
      v                                         v
+------------------+                            +------------------------+
| Order gRPC Server|                            |      RabbitMQ          |
|     (:9090)      |                            |   (durable exchange)   |
+------------------+                            +------------------------+
      |                                                 |
      | server-side streaming                           | (payment.completed queue)
      v                                                 v
Subscriber (client)                            +------------------------+
                                               | Notification Service   |
                                               | (manual ACK,           |
                                               |  idempotency check)    |
                                               +------------------------+
                                                         |
                                                         v
                                                  notification_db
                                                  (processed_events)
```

**Key Changes from Assignment 2:**
- Notification Service added as a new microservice
- RabbitMQ message broker for event-driven communication
- Payment events published after successful payment transaction
- Manual ACKs ensure at-least-once delivery
- Idempotency check prevents duplicate notifications
- customer_email field added to PaymentRequest proto for proper email routing

---

## Event Flow

1. Client creates order via Order Service REST API
2. Order Service calls Payment Service via gRPC
3. Payment Service saves payment to database
4. Payment Service publishes `payment.completed` event to RabbitMQ
5. Notification Service consumes event from queue
6. Notification Service checks idempotency (skip if already processed)
7. Notification Service logs email notification and ACKs message

---

## Idempotency Strategy

The Notification Service implements idempotency using a PostgreSQL table `processed_events`:

```sql
CREATE TABLE processed_events (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(255) UNIQUE NOT NULL,
    processed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
```

**Logic:**
1. Each payment event includes a unique `event_id` (UUID)
2. On message receipt, the consumer attempts to insert the `event_id`
3. Uses `ON CONFLICT (event_id) DO NOTHING` for atomic deduplication
4. If insert returns no rows, the event was already processed → skip
5. If insert succeeds, process the notification

This ensures that even if RabbitMQ delivers the same message twice (at-least-once semantics), the notification is only logged once.

---

## ACK Logic Implementation

**Consumer Configuration:**
- `autoAck: false` — Manual acknowledgment enabled
- `QoS prefetch: 1` — Only one unacknowledged message at a time
- Durable queue — Messages survive broker restarts
- Persistent messages — Delivery mode set to `Persistent` (2)

**Processing Flow:**
```
1. Receive message from queue
2. Unmarshal JSON payload
3. Check idempotency store
4. If duplicate: ACK immediately (message consumed but no action)
5. If new: Log notification
6. On success: ACK the message
7. On error: NACK with requeue=true (retry)
```

**Graceful Shutdown:**
- Consumer listens for SIGINT/SIGTERM
- Stops consuming new messages
- Closes RabbitMQ connection
- In-flight messages are requeued automatically

---

## Project Layout

```
.
├── docker-compose.yml
├── docker/postgres/init-multiple-dbs.sh
├── proto-gen/                        # generated code module
│   ├── go.mod
│   ├── payment/v1/
│   └── order/v1/
├── order-service/
│   ├── cmd/app/main.go
│   └── internal/
│       ├── api/          HTTP handlers (Gin)
│       ├── config/
│       ├── domain/       Order, OrderRepository
│       ├── grpc/         payment_client.go, order_server.go
│       ├── repository/   order_repository.go
│       ├── service/      OrderService
│       └── migrations/   SQL migrations
├── payment-service/
│   ├── cmd/app/main.go
│   └── internal/
│       ├── api/          HTTP handlers (Gin)
│       ├── config/
│       ├── domain/       Payment, PaymentRepository
│       ├── grpc/         payment_server.go
│       ├── messaging/    Event publisher interface + RabbitMQ impl
│       ├── repository/   payment_repository.go
│       ├── service/      PaymentService (publishes events)
│       └── migrations/   SQL migrations
└── notification-service/
    ├── cmd/app/main.go
    └── internal/
        ├── config/       Environment configuration
        ├── domain/       PaymentEvent, IdempotencyStore
        ├── messaging/    Event consumer interface + RabbitMQ impl
        ├── repository/   Idempotency repository (PostgreSQL)
        ├── service/      NotificationService (logs emails)
        └── migrations/   SQL migrations
```

---

## Environment Variables

### payment-service

| Variable            | Default          | Description                |
| ------------------- | ---------------- | -------------------------- |
| `PORT`              | `8081`           | HTTP server port           |
| `GRPC_HOST`         | `0.0.0.0`        | gRPC server host           |
| `GRPC_PORT`         | `9091`           | gRPC server port           |
| `DB_HOST`           | `localhost`      | Postgres host              |
| `DB_PORT`           | `5432`            | Postgres port              |
| `DB_USER`           | `postgres`       | Postgres user              |
| `DB_PASSWORD`       | —                | Postgres password          |
| `DB_NAME`           | `payment_db`     | Database name              |
| `DB_SSLMODE`        | `disable`        | SSL mode                   |
| `RABBITMQ_HOST`     | `localhost`      | RabbitMQ host              |
| `RABBITMQ_PORT`     | `5672`           | RabbitMQ AMQP port         |
| `RABBITMQ_USER`     | `rabbitmq`       | RabbitMQ username          |
| `RABBITMQ_PASSWORD` | `rabbitmq123`    | RabbitMQ password          |

### order-service

| Variable               | Default          | Description                  |
| ---------------------- | ---------------- | ---------------------------- |
| `PORT`                 | `8080`           | HTTP server port             |
| `GRPC_HOST`            | `0.0.0.0`        | gRPC server host             |
| `ORDER_GRPC_PORT`      | `9090`           | gRPC streaming server port   |
| `PAYMENT_GRPC_ADDRESS` | `payment-service:9091` | Payment service gRPC address |
| `DB_HOST`              | `localhost`      | Postgres host                |
| `DB_PORT`              | `5432`           | Postgres port                |
| `DB_USER`              | `postgres`       | Postgres user                |
| `DB_PASSWORD`          | —                | Postgres password            |
| `DB_NAME`              | `order_db`       | Database name                |
| `DB_SSLMODE`           | `disable`        | SSL mode                     |

### notification-service

| Variable            | Default          | Description                |
| ------------------- | ---------------- | -------------------------- |
| `DB_HOST`           | `localhost`      | Postgres host              |
| `DB_PORT`           | `5432`           | Postgres port              |
| `DB_USER`           | `postgres`       | Postgres user              |
| `DB_PASSWORD`       | —                | Postgres password          |
| `DB_NAME`           | `notification_db`| Database name              |
| `DB_SSLMODE`        | `disable`        | SSL mode                   |
| `RABBITMQ_HOST`     | `localhost`      | RabbitMQ host              |
| `RABBITMQ_PORT`     | `5672`           | RabbitMQ AMQP port         |
| `RABBITMQ_USER`     | `rabbitmq`       | RabbitMQ username          |
| `RABBITMQ_PASSWORD` | `rabbitmq123`    | RabbitMQ password          |

---

## Run with Docker

1. Start all services:

```bash
docker compose up --build
```

2. Test the flow:

```bash
# Create an order (this triggers payment and notification)
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"c1","customer_email":"test@example.com","item_name":"Book","amount":1500}'

# Check notification service logs
docker logs notification-service
```

Expected notification log:
```
[Notification] Sent email to test@example.com for Order #<order-id>. Amount: $15.00
```

---

## REST API

### Order Service

```bash
curl http://localhost:8080/health

curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"c1","customer_email":"test@example.com","item_name":"Book","amount":1500}'

curl http://localhost:8080/orders/<id>

curl -X POST http://localhost:8080/orders/<id>/cancel
```

### Payment Service

```bash
curl http://localhost:8081/health
curl http://localhost:8081/payments/<id>
```

Payment rule: `amount <= 100000` → `Authorized`; `amount > 100000` → `Declined`.

---

## Reliability Guarantees

| Guarantee          | Implementation                                    |
| ------------------ | ------------------------------------------------- |
| At-least-once      | Manual ACKs, durable queues, persistent messages  |
| Idempotency        | PostgreSQL unique constraint on `event_id`        |
| Graceful Shutdown  | os/signal handling, context cancellation          |
| Message Persistence| Queue survives broker restart                     |

---

## Notes

- RabbitMQ management UI available at `http://localhost:15672` (user: `rabbitmq`, pass: `rabbitmq123`)
- Proto files are maintained in a separate repository and generated code is included via local replace directive
