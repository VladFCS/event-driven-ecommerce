# Roadmap

This roadmap is the path from the current synchronous prototype to a project that can honestly be presented as an **event-driven ecommerce app**.

Status markers:
- `[x]` done
- `[ ]` not done yet
- `[~]` partially done / in progress

## Phase 1: Showcase-ready foundation

### 1. Add automated tests
- [x] Add unit tests for `order-service`
- [ ] Add unit tests for `inventory-service`
- [ ] Add payment idempotency tests
- [ ] Add gateway handler tests for key endpoints
- [ ] Add checkout compensation tests

### 2. Improve developer workflow
- [ ] Add `make lint`
- [x] Add `make proto`
- [x] Add `make build`
- [x] Add `make test`
- [x] Add local run instructions for all services

### 3. Add local infrastructure
- [~] Add `docker-compose.yml` (implemented as `compose.yaml`)
- [x] Add Kafka container
- [x] Add PostgreSQL container
- [ ] Add MongoDB container
- [x] Add Redis container
- [x] Add optional Kafka UI

### 4. Improve project presentation
- [x] Add architecture diagram to README
- [~] Add sample requests and responses
- [x] Add service responsibility table
- [x] Add note about what is implemented today vs target architecture

## Phase 2: Move core state out of memory

### 5. Replace memory repositories
- [x] Move orders to PostgreSQL
- [x] Move payments to PostgreSQL
- [x] Move inventory to PostgreSQL
- [x] Move catalog to MongoDB

### 6. Introduce Redis where it adds value
- [ ] Idempotency key storage
- [ ] Request/result caching where appropriate
- [ ] Rate limiting support in gateway

## Phase 3: Introduce real event-driven flow

### 7. Define event contracts
- [x] Create event envelope
- [x] Add `event_id`
- [x] Add `event_type`
- [~] Add `aggregate_id`
- [x] Add `occurred_at`
- [ ] Add `correlation_id`
- [ ] Add `request_id`
- [ ] Add payload versioning

### 8. Define first business events
- [x] `order.created`
- [x] `inventory.reserved`
- [~] `inventory.failed` (implemented as `inventory.reservation_failed`)
- [x] `payment.created`
- [x] `payment.captured`
- [x] `payment.failed`
- [x] `order.cancelled`

### 9. Add Kafka producer/consumer infrastructure
- [x] Producer abstraction
- [~] Consumer abstraction
- [x] Topic naming convention
- [~] Retry policy
- [x] Consumer logging

## Phase 4: Build first asynchronous workflow

### 10. Convert checkout into a real async flow
- [x] Gateway submits checkout command
- [x] `order-service` creates order and publishes `order.created`
- [x] `inventory-service` consumes `order.created`
- [x] Inventory publishes success or failure
- [x] `payment-service` consumes `inventory.reserved`
- [x] Payment publishes success or failure
- [x] Payment outcome can be triggered manually for local demo flow
- [x] `order-service` consumes final outcome and updates order state

### 11. Define order lifecycle states
- [x] `PENDING`
- [x] `AWAITING_PAYMENT`
- [x] `CONFIRMED`
- [ ] `FAILED`
- [x] `CANCELLED`

Current implementation note:
- `AWAITING_PAYMENT` is the current intermediate order state
- separate `AWAITING_INVENTORY`, `INVENTORY_RESERVED`, and `PAYMENT_PENDING` states are not implemented

## Phase 5: Make event processing production-minded

### 12. Add idempotent consumers
- [x] Prevent duplicate processing
- [x] Track processed events
- [x] Make handlers replay-safe

### 13. Add outbox pattern
- [~] Store business state and event in one transaction
- [~] Publish from outbox worker
- [~] Handle retry and publish recovery

Current implementation note:
- outbox is implemented for `order-service`
- inventory and payment publishers are still direct Kafka writes
- `payment-service` still publishes outcome events directly after DB update

### 14. Add failure handling
- [~] Retry transient failures
- [x] Add dead-letter topic or failure topic
- [x] Add compensating event paths

## Phase 6: Observability and operations

### 15. Add metrics
- [x] Requests by endpoint
- [ ] gRPC call failures
- [ ] Events published
- [ ] Events consumed
- [ ] Consumer retries
- [ ] DLQ counts
- [~] Checkout latency

### 16. Improve structured logging
- [x] Include `request_id`
- [ ] Include `correlation_id`
- [~] Include `order_id`
- [~] Include event type and handler name

### 17. Add tracing mindset
- [ ] Propagate correlation IDs
- [ ] Trace synchronous and asynchronous boundaries

### 18. Add health/readiness improvements
- [x] Readiness based on dependency availability
- [ ] Kafka connectivity checks
- [ ] Consumer health visibility

## Phase 7: Delivery and deployment

### 19. Add CI
- [ ] Run `go test ./...`
- [ ] Add linter
- [ ] Build binaries
- [ ] Fail on generated code drift if needed

### 20. Add containerization
- [ ] Dockerfiles for all services
- [x] Compose-based local startup

### 21. Add Kubernetes manifests
- [ ] Deployments
- [ ] Services
- [ ] ConfigMaps
- [ ] Secrets strategy
- [ ] Probes

## Recommended implementation order

If you want the shortest path to a believable event-driven demo, do these first:

1. Add tests
2. Add Docker Compose
3. Move orders and payments to PostgreSQL
4. Introduce Kafka
5. Implement `order.created`
6. Make `inventory-service` react to that event
7. Make `payment-service` react to inventory success
8. Make `order-service` finalize order status from events
9. Add idempotent consumers
10. Add outbox pattern

## Definition of done for “event-driven ecommerce”

The project can honestly use that name when:
- events drive business progress
- services consume and publish domain events
- order state advances because of events, not only gateway RPC orchestration
- duplicate events are handled safely
- failure paths are documented and observable
