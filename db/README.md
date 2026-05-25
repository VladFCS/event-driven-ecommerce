# Database workflow

This repo uses `golang-migrate` for PostgreSQL schema changes.

## Shared local PostgreSQL workflow

Local default database settings come from the `Makefile`:

- host: `localhost`
- port: `5432`
- database: `ecommerce`
- user: `app`
- password: `app`

Derived DSN:

```text
postgres://app:app@localhost:5432/ecommerce?sslmode=disable
```

Local `payment-service` and `order-service` currently share:

- one PostgreSQL database: `ecommerce`
- one migration directory: `db/migrations`
- one sequential migration history

Current shared migration sequence:

- `000001_create_payments`
- `000002_create_orders`

## Commands

Install the migration tool if needed:

```bash
make migrate-install
```

Start PostgreSQL and apply shared migrations:

```bash
make db-prepare
```

Start PostgreSQL only:

```bash
make db-up
```

Apply shared migrations only:

```bash
make db-migrate-up
```

Roll back the last shared migration:

```bash
make db-migrate-down
```

Show the current shared migration version:

```bash
make db-migrate-version
```

Typical local sequence:

```bash
make db-up
make db-migrate-up
make run-payment
make run-order
```

Or use the combined setup target:

```bash
make db-prepare
```

Run `payment-service` against Postgres:

```bash
make run-payment
```

Run `order-service` against Postgres:

```bash
make run-order
```
