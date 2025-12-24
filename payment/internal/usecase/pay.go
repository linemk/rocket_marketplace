package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/linemk/rocket-shop/payment/internal/entyties/apperrors"
	"github.com/linemk/rocket-shop/payment/internal/entyties/models"
	"github.com/linemk/rocket-shop/platform/pkg/logger"
	payment_v1 "github.com/linemk/rocket-shop/shared/pkg/proto/payment/v1"
)

func (uc *useCase) PayOrder(ctx context.Context, orderUUID, userID string, paymentMethod payment_v1.PaymentMethod) (string, error) {
	// Валидация входных данных
	if orderUUID == "" {
		return "", apperrors.ErrInvalidAmount
	}
	if userID == "" {
		return "", apperrors.ErrInvalidAmount
	}

	// Генерируем UUID транзакции
	transactionUUID := uuid.New()
	now := time.Now()

	// Создаем транзакцию (amount будет 0, так как его нет в protobuf)
	transaction := models.Transaction{
		UUID:          transactionUUID.String(),
		OrderUUID:     orderUUID,
		UserID:        userID,
		PaymentMethod: paymentMethod,
		Amount:        0,                                 // В реальном приложении amount должен приходить извне
		Status:        models.TransactionStatusCompleted, // В реальном приложении здесь была бы логика обработки платежа
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Сохраняем транзакцию в репозитории
	if err := uc.paymentRepository.CreateTransaction(ctx, transaction); err != nil {
		return "", apperrors.ErrPaymentFailed
	}

	// Выводим сообщение в консоль согласно спецификации
	logger.Info(ctx, "Оплата прошла успешно", zap.String("transaction_uuid", transactionUUID.String()))

	return transactionUUID.String(), nil
}
