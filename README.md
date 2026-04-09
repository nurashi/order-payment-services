# Microservices: Order & Payment (gRPC Migration)

Assignment 2 — gRPC Migration & Contract-First Development.

Proto repository: https://github.com/nurashi/ap2-protos  
Generated code repository: https://github.com/nurashi/ap2-proto-gen

---

## Architecture

```
External Client
      |
      | REST (HTTP :8080)
      v
 Order Service ──── gRPC (:9091) ────> Payment Service
      |                                      |
      | pg_notify order_updates              |
      v                                      v
  order_db                             payment_db
      |
      | PostgreSQL LISTEN/NOTIFY
      v
 Order gRPC Server (:9090)
      |
      | server-side streaming
      v
 Subscriber (client)
```

**Inter-service communication** changed from HTTP/JSON to gRPC.  
**Order Service** still exposes REST endpoints for external consumers (Gin).  
**Payment Service** runs both an HTTP server (port 8081) and a gRPC server (port 9091) simultaneously.  
**Order Service** runs both an HTTP server (port 8080) and a gRPC server (port 9090) simultaneously.

---

## Contract-First Workflow

Proto files live inside each service under `proto/`:

```
payment-service/proto/payment/v1/payment.proto
order-service/proto/order/v1/order.proto
```

Generated Go code lives in `proto-gen/` (separate module `github.com/nurashi/ap2-proto-gen`).

To regenerate after changing a `.proto` file:

```bash
# from payment-service/
protoc \
  --proto_path=proto \
  --proto_path=/usr/include \
  --go_out=../proto-gen \
  --go_opt=paths=source_relative \
  --go-grpc_out=../proto-gen \
  --go-grpc_opt=paths=source_relative \
  payment/v1/payment.proto

# from order-service/
protoc \
  --proto_path=proto \
  --proto_path=/usr/include \
  --go_out=../proto-gen \
  --go_opt=paths=source_relative \
  --go-grpc_out=../proto-gen \
  --go-grpc_opt=paths=source_relative \
  order/v1/order.proto
```

Both services reference `proto-gen/` via a `replace` directive in `go.mod` for local development:

```
replace github.com/nurashi/ap2-proto-gen => ../proto-gen
```

---

## gRPC Services

### PaymentService (`payment-service/proto/payment/v1/payment.proto`)

```protobuf
service PaymentService {
  rpc ProcessPayment(PaymentRequest) returns (PaymentResponse);
}
```

### OrderService (`order-service/proto/order/v1/order.proto`)

```protobuf
service OrderService {
  rpc SubscribeToOrderUpdates(OrderRequest) returns (stream OrderStatusUpdate);
}
```

The streaming endpoint is backed by PostgreSQL `LISTEN`/`NOTIFY`. When an order status changes in the database, `pg_notify('order_updates', '<id>:<status>')` fires and the stream pushes the update to the subscriber immediately.

---

## Project Layout

```
.
├── docker-compose.postgres.yml
├── docker/postgres/init-multiple-dbs.sh
├── proto-gen/                        # generated code module
│   ├── go.mod
│   ├── payment/v1/
│   └── order/v1/
├── order-service/
│   ├── proto/order/v1/order.proto
│   ├── cmd/app/main.go
│   └── internal/
│       ├── api/          HTTP handlers (Gin)
│       ├── config/
│       ├── domain/       Order, OrderRepository, OrderSubscriber
│       ├── grpc/         payment_client.go, order_server.go
│       ├── repository/   order_repository.go, order_subscriber.go
│       └── service/      OrderService use case
└── payment-service/
    ├── proto/payment/v1/payment.proto
    ├── cmd/app/main.go
    └── internal/
        ├── api/          HTTP handlers (Gin)
        ├── config/
        ├── domain/
        ├── grpc/         payment_server.go, interceptors.go
        ├── repository/
        └── service/      PaymentService use case
```

---

## Environment Variables

### payment-service

| Variable    | Default | Description              |
|-------------|---------|--------------------------|
| `PORT`      | `8081`  | HTTP server port         |
| `GRPC_PORT` | `9091`  | gRPC server port         |
| `DB_HOST`   | `localhost` | Postgres host        |
| `DB_PORT`   | `5432`  | Postgres port            |
| `DB_USER`   | `postgres` | Postgres user         |
| `DB_PASSWORD` | —     | Postgres password        |
| `DB_NAME`   | `payment_db` | Database name       |
| `DB_SSLMODE` | `disable` | SSL mode             |

### order-service

| Variable               | Default          | Description                       |
|------------------------|------------------|-----------------------------------|
| `PORT`                 | `8080`           | HTTP server port                  |
| `ORDER_GRPC_PORT`      | `9090`           | gRPC streaming server port        |
| `PAYMENT_GRPC_ADDRESS` | `localhost:9091` | Payment service gRPC address      |
| `DB_HOST`              | `localhost`      | Postgres host                     |
| `DB_PORT`              | `5432`           | Postgres port                     |
| `DB_USER`              | `postgres`       | Postgres user                     |
| `DB_PASSWORD`          | —                | Postgres password                 |
| `DB_NAME`              | `order_db`       | Database name                     |
| `DB_SSLMODE`           | `disable`        | SSL mode                          |

---

## Run Locally

1. Start Postgres:

```bash
docker compose -f docker-compose.postgres.yml up -d
```

2. Start payment service:

```bash
cd payment-service
go run ./cmd/app
```

Starts HTTP on `:8081` and gRPC on `:9091`.

3. Start order service in a separate terminal:

```bash
cd order-service
go run ./cmd/app
```

Starts HTTP on `:8080`, gRPC streaming on `:9090`.

Migrations run automatically on startup.

---

## REST API (unchanged from Assignment 1)

### Order Service

```bash
curl http://localhost:8080/health

curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"c1","item_name":"Book","amount":1500}'

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

## Notes

- The `PaymentClient` interface in `order-service/internal/service/order_service.go` is unchanged from Assignment 1. Only the concrete implementation changed from HTTP to gRPC.
- The `PaymentService` use case in payment-service is unchanged. The gRPC server is an additional delivery layer.
- The logging interceptor on the payment gRPC server logs every incoming method name and duration.
- Real-time streaming is driven by PostgreSQL `NOTIFY`, not by polling or `time.Sleep`.
