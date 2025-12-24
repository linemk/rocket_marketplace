package order_consumer

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/linemk/rocket-shop/assembly/internal/converter/kafka/decoder"
	"github.com/linemk/rocket-shop/assembly/internal/metrics"
	"github.com/linemk/rocket-shop/assembly/internal/model"
	"github.com/linemk/rocket-shop/assembly/internal/service"
	platformKafka "github.com/linemk/rocket-shop/platform/pkg/kafka"
)

type handler struct {
	orderProducer service.OrderProducerService
	metrics       *metrics.AssemblyMetrics
	logger        Logger
}

func NewHandler(orderProducer service.OrderProducerService, metrics *metrics.AssemblyMetrics, logger Logger) HandlerFunc {
	h := &handler{
		orderProducer: orderProducer,
		metrics:       metrics,
		logger:        logger,
	}

	return h.Handle
}

func (h *handler) Handle(ctx context.Context, msg platformKafka.Message) error {
	h.logger.Info(ctx, "Received OrderPaid event", zap.String("topic", msg.Topic))

	// Декодируем событие
	event, err := decoder.DecodeOrderPaid(msg.Value)
	if err != nil {
		h.logger.Error(ctx, "Failed to decode OrderPaid event", zap.Error(err))
		return err
	}

	h.logger.Info(ctx, "Processing order assembly",
		zap.String("order_uuid", event.OrderUUID),
		zap.String("user_uuid", event.UserUUID),
	)

	// Симулируем сборку корабля (от 1 до 10 секунд)
	//nolint:gosec // G404: используется для симуляции, не для криптографии
	buildTimeSec := rand.Int64N(10) + 1

	h.logger.Info(ctx, "Starting ship assembly",
		zap.String("order_uuid", event.OrderUUID),
		zap.Int64("build_time_sec", buildTimeSec),
	)

	// Начинаем измерение времени сборки
	startTime := time.Now()
	status := "success"
	defer func() {
		duration := time.Since(startTime).Seconds()
		h.metrics.Duration.WithLabelValues(status).Observe(duration)
	}()

	// Ждем buildTimeSec секунд через таймер с контекстом
	timer := time.NewTimer(time.Duration(buildTimeSec) * time.Second)
	defer timer.Stop()

	select {
	case <-timer.C:
		// Сборка завершена
	case <-ctx.Done():
		status = "error"
		return ctx.Err()
	}

	// Отправляем событие ShipAssembled
	shipAssembledEvent := &model.ShipAssembledEvent{
		EventUUID:    uuid.New().String(),
		OrderUUID:    event.OrderUUID,
		UserUUID:     event.UserUUID,
		BuildTimeSec: buildTimeSec,
	}

	if err := h.orderProducer.SendShipAssembled(ctx, shipAssembledEvent); err != nil {
		h.logger.Error(ctx, "Failed to send ShipAssembled event", zap.Error(err))
		return err
	}

	h.logger.Info(ctx, "Ship assembly completed successfully",
		zap.String("order_uuid", event.OrderUUID),
		zap.Int64("build_time_sec", buildTimeSec),
	)

	return nil
}
