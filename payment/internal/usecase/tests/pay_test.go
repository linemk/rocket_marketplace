package tests

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/linemk/rocket-shop/payment/internal/entyties/apperrors"
	"github.com/linemk/rocket-shop/payment/internal/mocks"
	"github.com/linemk/rocket-shop/payment/internal/usecase"
	"github.com/linemk/rocket-shop/platform/pkg/logger"
	payment_v1 "github.com/linemk/rocket-shop/shared/pkg/proto/payment/v1"
)

func TestPayOrder(t *testing.T) {
	ctx := context.Background()
	// Инициализируем logger для тестов
	if err := logger.Init(ctx, "info", false, false, "", "payment-test"); err != nil {
		t.Fatalf("failed to init logger: %v", err)
	}

	type fields struct {
		repoMock func() *mocks.MockPaymentRepository
	}

	type args struct {
		orderUUID     string
		userID        string
		paymentMethod payment_v1.PaymentMethod
	}

	orderUUID := uuid.New().String()
	userID := uuid.New().String()

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "successfully pay order",
			fields: fields{
				repoMock: func() *mocks.MockPaymentRepository {
					mockRepo := mocks.NewMockPaymentRepository(gomock.NewController(t))
					mockRepo.EXPECT().CreateTransaction(ctx, gomock.Any()).Return(nil)
					return mockRepo
				},
			},
			args: args{
				orderUUID:     orderUUID,
				userID:        userID,
				paymentMethod: payment_v1.PaymentMethod_PAYMENT_METHOD_CARD,
			},
			wantErr: false,
		},
		{
			name: "pay order with empty orderUUID",
			fields: fields{
				repoMock: func() *mocks.MockPaymentRepository {
					return mocks.NewMockPaymentRepository(gomock.NewController(t))
				},
			},
			args: args{
				orderUUID:     "",
				userID:        userID,
				paymentMethod: payment_v1.PaymentMethod_PAYMENT_METHOD_CARD,
			},
			wantErr: true,
		},
		{
			name: "pay order with empty userID",
			fields: fields{
				repoMock: func() *mocks.MockPaymentRepository {
					return mocks.NewMockPaymentRepository(gomock.NewController(t))
				},
			},
			args: args{
				orderUUID:     orderUUID,
				userID:        "",
				paymentMethod: payment_v1.PaymentMethod_PAYMENT_METHOD_CARD,
			},
			wantErr: true,
		},
		{
			name: "pay order with repository error",
			fields: fields{
				repoMock: func() *mocks.MockPaymentRepository {
					mockRepo := mocks.NewMockPaymentRepository(gomock.NewController(t))
					mockRepo.EXPECT().CreateTransaction(ctx, gomock.Any()).Return(apperrors.ErrPaymentFailed)
					return mockRepo
				},
			},
			args: args{
				orderUUID:     orderUUID,
				userID:        userID,
				paymentMethod: payment_v1.PaymentMethod_PAYMENT_METHOD_CARD,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paymentRepo := tt.fields.repoMock()
			uc := usecase.NewUseCase(paymentRepo)

			transactionUUID, err := uc.PayOrder(ctx, tt.args.orderUUID, tt.args.userID, tt.args.paymentMethod)

			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, transactionUUID)
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, transactionUUID)
		})
	}
}
