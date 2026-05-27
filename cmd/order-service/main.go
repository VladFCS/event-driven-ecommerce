package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	orderv1 "github.com/vladfc/event-driven-ecommerce-app/gen/order/v1"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/consumers"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/events"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/handler"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/repository"
	"github.com/vladfc/event-driven-ecommerce-app/internal/order/service"
	"github.com/vladfc/event-driven-ecommerce-app/internal/shared/grpcmiddleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	log := NewLogger("order-service")
	grpcPort := getenv("GRPC_PORT", "50054")

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Error("failed to listen", slog.Any("error", err))
		os.Exit(1)
	}

	orderCreatedTopic := strings.TrimSpace(getenv("KAFKA_ORDER_CREATED_TOPIC", "orders.created"))
	if orderCreatedTopic == "" {
		log.Error("failed to configure order event publisher", slog.Any("error", errors.New("KAFKA_ORDER_CREATED_TOPIC is required")))
		os.Exit(1)
	}
	orderCancelledTopic := strings.TrimSpace(getenv("KAFKA_ORDER_CANCELLED_TOPIC", "order.cancelled"))
	if orderCancelledTopic == "" {
		log.Error("failed to configure order event publisher", slog.Any("error", errors.New("KAFKA_ORDER_CANCELLED_TOPIC is required")))
		os.Exit(1)
	}

	pool, closePool, err := newOrderDatabasePool(log)
	if err != nil {
		log.Error("failed to configure order repository", slog.Any("error", err))
		os.Exit(1)
	}
	defer closePool()

	orderRepository := repository.NewPostgresRepository(pool, orderCreatedTopic, orderCancelledTopic)

	publisher, closePublisher, err := newOrderEventPublisher(log)
	if err != nil {
		log.Error("failed to configure order event publisher", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := closePublisher(); err != nil {
			log.Error("failed to close order event publisher", slog.Any("error", err))
		}
	}()

	deduplicator, closeDeduplicator, err := newOrderEventDeduplicator(log)
	if err != nil {
		log.Error("failed to configure order event deduplicator", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := closeDeduplicator(); err != nil {
			log.Error("failed to close order event deduplicator", slog.Any("error", err))
		}
	}()

	outboxConfig, err := newOutboxDispatcherConfig()
	if err != nil {
		log.Error("failed to configure order outbox dispatcher", slog.Any("error", err))
		os.Exit(1)
	}
	log.Info(
		"configured order outbox dispatcher",
		slog.String("poll_interval", outboxConfig.PollInterval.String()),
		slog.String("lock_timeout", outboxConfig.LockTimeout.String()),
		slog.String("publish_timeout", outboxConfig.PublishTimeout.String()),
		slog.Int("batch_size", int(outboxConfig.BatchSize)),
	)

	service := service.NewOrderService(orderRepository)
	grpcHandler := handler.NewGRPCHandler(service, log)

	inventoryReservedConsumer, closeInventoryReservedConsumer, err := newInventoryReservedConsumer(service, deduplicator, log)
	if err != nil {
		log.Error("failed to configure inventory.reserved consumer", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := closeInventoryReservedConsumer(); err != nil {
			log.Error("failed to close inventory.reserved consumer", slog.Any("error", err))
		}
	}()

	inventoryReservationFailedConsumer, closeInventoryReservationFailedConsumer, err := newInventoryReservationFailedConsumer(service, deduplicator, log)
	if err != nil {
		log.Error("failed to configure inventory.reservation_failed consumer", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := closeInventoryReservationFailedConsumer(); err != nil {
			log.Error("failed to close inventory.reservation_failed consumer", slog.Any("error", err))
		}
	}()

	paymentCapturedConsumer, closePaymentCapturedConsumer, err := newPaymentCapturedConsumer(service, deduplicator, log)
	if err != nil {
		log.Error("failed to configure payment.captured consumer", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := closePaymentCapturedConsumer(); err != nil {
			log.Error("failed to close payment.captured consumer", slog.Any("error", err))
		}
	}()

	paymentFailedConsumer, closePaymentFailedConsumer, err := newPaymentFailedConsumer(service, deduplicator, log)
	if err != nil {
		log.Error("failed to configure payment.failed consumer", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := closePaymentFailedConsumer(); err != nil {
			log.Error("failed to close payment.failed consumer", slog.Any("error", err))
		}
	}()

	paymentCreationFailedConsumer, closePaymentCreationFailedConsumer, err := newPaymentCreationFailedConsumer(service, deduplicator, log)
	if err != nil {
		log.Error("failed to configure payment.creation_failed consumer", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := closePaymentCreationFailedConsumer(); err != nil {
			log.Error("failed to close payment.creation_failed consumer", slog.Any("error", err))
		}
	}()

	server := grpc.NewServer(
		grpc.UnaryInterceptor(grpcmiddleware.RequestIDUnaryServerInterceptor()),
	)
	orderv1.RegisterOrderServiceServer(server, grpcHandler)

	reflection.Register(server)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	outboxDispatcher := events.NewOutboxDispatcher(pool, publisher, log, outboxConfig)
	serverErrCh := make(chan error, 1)
	workerErrCh := make(chan error, 1)
	consumerErrCh := make(chan error, 5)

	go func() {
		if err := outboxDispatcher.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			workerErrCh <- err
			return
		}
		if ctx.Err() == nil {
			workerErrCh <- errors.New("outbox dispatcher stopped unexpectedly")
		}
	}()

	go func() {
		log.Info("order-service started", slog.String("grpc_port", grpcPort))
		if serveErr := server.Serve(lis); serveErr != nil {
			serverErrCh <- serveErr
		}
	}()

	startConsumer := func(name string, consumer *consumers.TopicConsumer) {
		go func() {
			log.Info("starting consumer", slog.String("consumer", name))
			if runErr := consumer.Run(ctx); runErr != nil && ctx.Err() == nil {
				consumerErrCh <- fmt.Errorf("%s consumer: %w", name, runErr)
			}
		}()
	}

	startConsumer("inventory.reserved", inventoryReservedConsumer)
	startConsumer("inventory.reservation_failed", inventoryReservationFailedConsumer)
	startConsumer("payment.captured", paymentCapturedConsumer)
	startConsumer("payment.failed", paymentFailedConsumer)
	startConsumer("payment.creation_failed", paymentCreationFailedConsumer)

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-serverErrCh:
		log.Error("grpc server stopped with error", slog.Any("error", err))
		stop()
	case err := <-workerErrCh:
		log.Error("outbox dispatcher stopped with error", slog.Any("error", err))
		stop()
	case err := <-consumerErrCh:
		log.Error("order event consumer stopped with error", slog.Any("error", err))
		stop()
	}

	log.Info("shutting down order-service")
	server.GracefulStop()
}

func NewLogger(serviceName string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})).With(
		slog.String("service", serviceName),
	)
}

func getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func newOrderDatabasePool(logger *slog.Logger) (*pgxpool.Pool, func(), error) {
	databaseURL := strings.TrimSpace(getenv("ORDER_DATABASE_URL", ""))
	if databaseURL == "" {
		return nil, nil, errors.New("ORDER_DATABASE_URL is required")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to postgres: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("ping postgres: %w", err)
	}

	logger.Info("configured postgres order repository")

	return pool, pool.Close, nil
}

func newOrderEventPublisher(logger *slog.Logger) (events.Publisher, func() error, error) {
	brokers := parseKafkaBrokers(getenv("KAFKA_BROKERS", ""))
	topic := strings.TrimSpace(getenv("KAFKA_ORDER_CREATED_TOPIC", "orders.created"))

	if len(brokers) == 0 {
		return nil, nil, errors.New("KAFKA_BROKERS is required")
	}

	if topic == "" {
		return nil, nil, errors.New("KAFKA_ORDER_CREATED_TOPIC is required")
	}

	publisher := events.NewKafkaPublisher(brokers, logger)
	logger.Info(
		"configured kafka order event publisher",
		slog.Any("brokers", brokers),
		slog.String("topic", topic),
	)
	return publisher, publisher.Close, nil
}

func newOrderEventDeduplicator(logger *slog.Logger) (consumers.EventDeduplicator, func() error, error) {
	addr := strings.TrimSpace(getenv("REDIS_ADDR", "localhost:6379"))
	password := getenv("REDIS_PASSWORD", "")
	rawDB := strings.TrimSpace(getenv("REDIS_DB", "0"))
	rawTTL := strings.TrimSpace(getenv("REDIS_EVENT_DEDUP_TTL", "24h"))

	if addr == "" {
		return nil, nil, errors.New("REDIS_ADDR is required")
	}

	db, err := strconv.Atoi(rawDB)
	if err != nil {
		return nil, nil, fmt.Errorf("parse REDIS_DB: %w", err)
	}

	ttl, err := time.ParseDuration(rawTTL)
	if err != nil {
		return nil, nil, fmt.Errorf("parse REDIS_EVENT_DEDUP_TTL: %w", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("ping redis: %w", err)
	}

	logger.Info(
		"configured redis order event deduplicator",
		slog.String("addr", addr),
		slog.Int("db", db),
		slog.String("ttl", ttl.String()),
	)

	return consumers.NewRedisEventDeduplicator(client, ttl), client.Close, nil
}

func parseKafkaBrokers(raw string) []string {
	parts := strings.Split(raw, ",")
	brokers := make([]string, 0, len(parts))
	for _, part := range parts {
		broker := strings.TrimSpace(part)
		if broker != "" {
			brokers = append(brokers, broker)
		}
	}
	return brokers
}

func newInventoryReservedConsumer(
	service *service.OrderService,
	deduplicator consumers.EventDeduplicator,
	logger *slog.Logger,
) (*consumers.TopicConsumer, func() error, error) {
	brokers := parseKafkaBrokers(getenv("KAFKA_BROKERS", ""))
	topic := strings.TrimSpace(getenv("KAFKA_INVENTORY_RESERVED_TOPIC", "inventory.reserved"))
	groupID := strings.TrimSpace(getenv("KAFKA_ORDER_CONSUMER_GROUP", "order-service"))

	if len(brokers) == 0 {
		return nil, nil, errors.New("KAFKA_BROKERS is required")
	}
	if topic == "" {
		return nil, nil, errors.New("KAFKA_INVENTORY_RESERVED_TOPIC is required")
	}
	if groupID == "" {
		return nil, nil, errors.New("KAFKA_ORDER_CONSUMER_GROUP is required")
	}

	consumer := consumers.NewInventoryReservedConsumer(brokers, topic, groupID, service, deduplicator, logger)
	return consumer, consumer.Close, nil
}

func newInventoryReservationFailedConsumer(
	service *service.OrderService,
	deduplicator consumers.EventDeduplicator,
	logger *slog.Logger,
) (*consumers.TopicConsumer, func() error, error) {
	brokers := parseKafkaBrokers(getenv("KAFKA_BROKERS", ""))
	topic := strings.TrimSpace(getenv("KAFKA_INVENTORY_RESERVATION_FAILED_TOPIC", "inventory.reservation_failed"))
	groupID := strings.TrimSpace(getenv("KAFKA_ORDER_CONSUMER_GROUP", "order-service"))

	if len(brokers) == 0 {
		return nil, nil, errors.New("KAFKA_BROKERS is required")
	}
	if topic == "" {
		return nil, nil, errors.New("KAFKA_INVENTORY_RESERVATION_FAILED_TOPIC is required")
	}
	if groupID == "" {
		return nil, nil, errors.New("KAFKA_ORDER_CONSUMER_GROUP is required")
	}

	consumer := consumers.NewInventoryReservationFailedConsumer(brokers, topic, groupID, service, deduplicator, logger)
	return consumer, consumer.Close, nil
}

func newPaymentCapturedConsumer(
	service *service.OrderService,
	deduplicator consumers.EventDeduplicator,
	logger *slog.Logger,
) (*consumers.TopicConsumer, func() error, error) {
	brokers := parseKafkaBrokers(getenv("KAFKA_BROKERS", ""))
	topic := strings.TrimSpace(getenv("KAFKA_PAYMENT_CAPTURED_TOPIC", "payment.captured"))
	groupID := strings.TrimSpace(getenv("KAFKA_ORDER_CONSUMER_GROUP", "order-service"))

	if len(brokers) == 0 {
		return nil, nil, errors.New("KAFKA_BROKERS is required")
	}
	if topic == "" {
		return nil, nil, errors.New("KAFKA_PAYMENT_CAPTURED_TOPIC is required")
	}
	if groupID == "" {
		return nil, nil, errors.New("KAFKA_ORDER_CONSUMER_GROUP is required")
	}

	consumer := consumers.NewPaymentCapturedConsumer(brokers, topic, groupID, service, deduplicator, logger)
	return consumer, consumer.Close, nil
}

func newPaymentFailedConsumer(
	service *service.OrderService,
	deduplicator consumers.EventDeduplicator,
	logger *slog.Logger,
) (*consumers.TopicConsumer, func() error, error) {
	brokers := parseKafkaBrokers(getenv("KAFKA_BROKERS", ""))
	topic := strings.TrimSpace(getenv("KAFKA_PAYMENT_FAILED_TOPIC", "payment.failed"))
	groupID := strings.TrimSpace(getenv("KAFKA_ORDER_CONSUMER_GROUP", "order-service"))

	if len(brokers) == 0 {
		return nil, nil, errors.New("KAFKA_BROKERS is required")
	}
	if topic == "" {
		return nil, nil, errors.New("KAFKA_PAYMENT_FAILED_TOPIC is required")
	}
	if groupID == "" {
		return nil, nil, errors.New("KAFKA_ORDER_CONSUMER_GROUP is required")
	}

	consumer := consumers.NewPaymentFailedConsumer(brokers, topic, groupID, service, deduplicator, logger)
	return consumer, consumer.Close, nil
}

func newPaymentCreationFailedConsumer(
	service *service.OrderService,
	deduplicator consumers.EventDeduplicator,
	logger *slog.Logger,
) (*consumers.TopicConsumer, func() error, error) {
	brokers := parseKafkaBrokers(getenv("KAFKA_BROKERS", ""))
	topic := strings.TrimSpace(getenv("KAFKA_PAYMENT_CREATION_FAILED_TOPIC", "payment.creation_failed"))
	groupID := strings.TrimSpace(getenv("KAFKA_ORDER_CONSUMER_GROUP", "order-service"))

	if len(brokers) == 0 {
		return nil, nil, errors.New("KAFKA_BROKERS is required")
	}
	if topic == "" {
		return nil, nil, errors.New("KAFKA_PAYMENT_CREATION_FAILED_TOPIC is required")
	}
	if groupID == "" {
		return nil, nil, errors.New("KAFKA_ORDER_CONSUMER_GROUP is required")
	}

	consumer := consumers.NewPaymentCreationFailedConsumer(brokers, topic, groupID, service, deduplicator, logger)
	return consumer, consumer.Close, nil
}

func newOutboxDispatcherConfig() (events.OutboxDispatcherConfig, error) {
	cfg := events.DefaultOutboxDispatcherConfig()

	pollInterval, err := parseDurationEnv("ORDER_OUTBOX_POLL_INTERVAL", cfg.PollInterval)
	if err != nil {
		return events.OutboxDispatcherConfig{}, err
	}
	lockTimeout, err := parseDurationEnv("ORDER_OUTBOX_LOCK_TIMEOUT", cfg.LockTimeout)
	if err != nil {
		return events.OutboxDispatcherConfig{}, err
	}
	publishTimeout, err := parseDurationEnv("ORDER_OUTBOX_PUBLISH_TIMEOUT", cfg.PublishTimeout)
	if err != nil {
		return events.OutboxDispatcherConfig{}, err
	}
	batchSize, err := parseInt32Env("ORDER_OUTBOX_BATCH_SIZE", cfg.BatchSize)
	if err != nil {
		return events.OutboxDispatcherConfig{}, err
	}

	cfg.PollInterval = pollInterval
	cfg.LockTimeout = lockTimeout
	cfg.PublishTimeout = publishTimeout
	cfg.BatchSize = batchSize

	if err := cfg.Validate(); err != nil {
		return events.OutboxDispatcherConfig{}, err
	}

	return cfg, nil
}

func parseDurationEnv(key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(getenv(key, fallback.String()))
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return value, nil
}

func parseInt32Env(key string, fallback int32) (int32, error) {
	raw := strings.TrimSpace(getenv(key, strconv.Itoa(int(fallback))))
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	if value > int(^uint32(0)>>1) {
		return 0, fmt.Errorf("parse %s: value %d exceeds int32 range", key, value)
	}
	return int32(value), nil
}
