# Database workflow

This repo uses `golang-migrate` for PostgreSQL schema changes.

## Payment service

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

## Commands

Install the migration tool if needed:

```bash
make migrate-install
```

Start PostgreSQL and apply migrations:

```bash
make db-prepare
```

Apply migrations only:

```bash
make db-migrate-up
```

Roll back the last migration:

```bash
make db-migrate-down
```

Show the current migration version:

```bash
make db-migrate-version
```

Run `payment-service` against Postgres:

```bash
make run-payment
```
