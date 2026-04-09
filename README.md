# Simple Microservices Demo

This project has two small Go services:

- `payment-service` handles payment authorization
- `order-service` creates orders and calls `payment-service`

Both services use the same Postgres container from the root of the project, but each service has its own database:

- `payment_db`
- `order_db`

## What it does

`payment-service` listens on `8081`.

- `POST /payments/process`
- `GET /payments/:id`
- `GET /health`

Payment rule is simple:

- if `amount <= 100000`, payment is authorized
- if `amount > 100000`, payment is declined

`order-service` listens on `8080`.

- `POST /orders`
- `GET /orders/:id`
- `POST /orders/:id/cancel`
- `GET /health`

Order flow is also simple:

- create order
- call `payment-service`
- save final order status as `Paid` or `Failed`

## Stack

- Go
- Gin
- PostgreSQL
- `pgx/v5`
- `goose` for migrations

## Project layout

```text
.
├── docker-compose.postgres.yml
├── docker/
│   └── postgres/
│       └── init-multiple-dbs.sh
├── order-service/
└── payment-service/
```

Each service has the usual structure inside:

- `cmd/app` for startup
- `internal/api` for handlers
- `internal/service` for business logic
- `internal/repository` for database access
- `internal/migrations` for SQL migrations

## Environment

`payment-service/.env`

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=123456
DB_NAME=payment_db
DB_SSLMODE=disable
```

`order-service/.env`

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=123456
DB_NAME=order_db
DB_SSLMODE=disable
```

`order-service` can also use:

```env
PAYMENT_SERVICE_URL=http://localhost:8081
PORT=8080
```

`payment-service` can also use:

```env
PORT=8081
```

If `PORT` or `PAYMENT_SERVICE_URL` are missing, defaults from code are used.

## Run locally

1. Start Postgres:

```bash
docker compose -f docker-compose.postgres.yml up -d
```

2. Start payment service:

```bash
cd payment-service
go run ./cmd/app
```

3. Start order service in another terminal:

```bash
cd order-service
go run ./cmd/app
```

Migrations run automatically on startup.

## Quick test

Check health:

```bash
curl http://localhost:8081/health
curl http://localhost:8080/health
```

Create order:

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"customer-1","item_name":"Book","amount":1500}'
```

Expected result: order should be created with status `Paid`.

Try declined payment:

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"customer-1","item_name":"Laptop","amount":150000}'
```

Expected result: order should be created with status `Failed`.

## Notes

- root Postgres is defined in `docker-compose.postgres.yml`
- the shell script in `docker/postgres` creates both databases on first container initialization
- if the Postgres data directory already exists, that script will not run again
