package usecase

import (
	"context"
	"fmt"

	uuidgen "github.com/google/uuid"

	"github.com/linemk/rocket-shop/order/internal/client/grpc/payment/converter"
	"github.com/linemk/rocket-shop/order/internal/entyties/apperrors"
	"github.com/linemk/rocket-shop/order/internal/entyties/events"
	"github.com/linemk/rocket-shop/order/internal/entyties/models"
	order_v1 "github.com/linemk/rocket-shop/shared/pkg/openapi/order/v1"
)

func (uc *useCase) PayOrder(ctx context.Context, uuid string, paymentMethod order_v1.PaymentMethod) (string, error) {
	// 1. Получаем заказ
	order, err := uc.orderRepository.Get(ctx, uuid)
	if err != nil {
		return "", apperrors.ErrOrderNotFound
	}

	// 2. Бизнес-логика: проверяем статус
	if order.Status != order_v1.OrderStatusPENDINGPAYMENT {
		return "", apperrors.ErrOrderCannotBePaid
	}

	// 3. Вызываем PaymentService
	protoPaymentMethod := converter.OpenAPIPaymentMethodToProto(paymentMethod)
	transactionUUID, err := uc.paymentClient.PayOrder(ctx, order.UUID, order.UserID, protoPaymentMethod)
	if err != nil {
		return "", apperrors.ErrPaymentFailed
	}

	// 4. Обновляем заказ
	updateInfo := models.OrderUpdateInfo{
		Status:        &[]order_v1.OrderStatus{order_v1.OrderStatusPAID}[0],
		TransactionID: &transactionUUID,
		PaymentMethod: &paymentMethod,
	}

	if err := uc.orderRepository.Update(ctx, uuid, updateInfo); err != nil {
		return "", fmt.Errorf("failed to update order: %w", err)
	}

	if uc.metrics != nil {
		uc.metrics.OrdersTotal.WithLabelValues("paid").Inc()
		uc.metrics.RevenueTotal.WithLabelValues(string(paymentMethod)).Add(float64(order.TotalPrice))
	}

	// 5. Отправляем событие OrderPaid
	event := &events.OrderPaidEvent{
		EventUUID:       uuidgen.New().String(),
		OrderUUID:       order.UUID,
		UserUUID:        order.UserID,
		PaymentMethod:   string(paymentMethod),
		TransactionUUID: transactionUUID,
	}

	if err := uc.orderProducerService.SendOrderPaid(ctx, event); err != nil {
		// Ошибка отправки события уже логируется в orderProducerService
		// Не прерываем основной процесс, т.к. заказ уже оплачен
		_ = err
	}

	return transactionUUID, nil
}
