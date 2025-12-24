package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/linemk/rocket-shop/order/internal/entyties/apperrors"
	"github.com/linemk/rocket-shop/order/internal/entyties/models"
	"github.com/linemk/rocket-shop/order/internal/mocks"
	"github.com/linemk/rocket-shop/order/internal/usecase"
	order_v1 "github.com/linemk/rocket-shop/shared/pkg/openapi/order/v1"
	payment_v1 "github.com/linemk/rocket-shop/shared/pkg/proto/payment/v1"
)

func TestPayOrder(t *testing.T) {
	ctx := context.Background()
	testUUID := uuid.New().String()
	transactionUUID := uuid.New().String()

	type fields struct {
		orderRepository      func() *mocks.MockOrderRepository
		paymentClient        func() *mocks.MockPaymentClient
		orderProducerService func() *mocks.MockOrderProducerService
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "successful pay order",
			fields: fields{
				orderRepository: func() *mocks.MockOrderRepository {
					mockClient := mocks.NewMockOrderRepository(gomock.NewController(t))
					mockClient.EXPECT().Get(ctx, testUUID).Return(models.Order{
						UUID:   testUUID,
						UserID: "user-123",
						Status: order_v1.OrderStatusPENDINGPAYMENT,
					}, nil)

					mockClient.EXPECT().Update(ctx, testUUID, gomock.Any()).Return(nil)

					return mockClient
				},
				paymentClient: func() *mocks.MockPaymentClient {
					mockClient := mocks.NewMockPaymentClient(gomock.NewController(t))
					mockClient.EXPECT().PayOrder(ctx, testUUID, "user-123", payment_v1.PaymentMethod_PAYMENT_METHOD_CARD).Return(transactionUUID, nil)

					return mockClient
				},
				orderProducerService: func() *mocks.MockOrderProducerService {
					mockService := mocks.NewMockOrderProducerService(gomock.NewController(t))
					mockService.EXPECT().SendOrderPaid(ctx, gomock.Any()).Return(nil)

					return mockService
				},
			},
			wantErr: false,
		},
		{
			name: "error order not found",
			fields: fields{
				orderRepository: func() *mocks.MockOrderRepository {
					mockClient := mocks.NewMockOrderRepository(gomock.NewController(t))
					mockClient.EXPECT().Get(ctx, testUUID).Return(models.Order{}, apperrors.ErrOrderNotFound)

					return mockClient
				},
				paymentClient: func() *mocks.MockPaymentClient {
					mockClient := mocks.NewMockPaymentClient(gomock.NewController(t))

					return mockClient
				},
				orderProducerService: func() *mocks.MockOrderProducerService {
					mockService := mocks.NewMockOrderProducerService(gomock.NewController(t))

					return mockService
				},
			},
			wantErr: true,
		},
		{
			name: "error order cannot be paid",
			fields: fields{
				orderRepository: func() *mocks.MockOrderRepository {
					mockClient := mocks.NewMockOrderRepository(gomock.NewController(t))
					mockClient.EXPECT().Get(ctx, testUUID).Return(models.Order{
						UUID:   testUUID,
						Status: order_v1.OrderStatusPAID,
					}, nil)

					return mockClient
				},
				paymentClient: func() *mocks.MockPaymentClient {
					mockClient := mocks.NewMockPaymentClient(gomock.NewController(t))

					return mockClient
				},
				orderProducerService: func() *mocks.MockOrderProducerService {
					mockService := mocks.NewMockOrderProducerService(gomock.NewController(t))

					return mockService
				},
			},
			wantErr: true,
		},
		{
			name: "error payment failed",
			fields: fields{
				orderRepository: func() *mocks.MockOrderRepository {
					mockClient := mocks.NewMockOrderRepository(gomock.NewController(t))
					mockClient.EXPECT().Get(ctx, testUUID).Return(models.Order{
						UUID:   testUUID,
						UserID: "user-123",
						Status: order_v1.OrderStatusPENDINGPAYMENT,
					}, nil)

					return mockClient
				},
				paymentClient: func() *mocks.MockPaymentClient {
					mockClient := mocks.NewMockPaymentClient(gomock.NewController(t))
					mockClient.EXPECT().PayOrder(ctx, testUUID, "user-123", payment_v1.PaymentMethod_PAYMENT_METHOD_CARD).Return("", fmt.Errorf("payment service error"))

					return mockClient
				},
				orderProducerService: func() *mocks.MockOrderProducerService {
					mockService := mocks.NewMockOrderProducerService(gomock.NewController(t))

					return mockService
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepository := tt.fields.orderRepository()
			paymentClient := tt.fields.paymentClient()
			orderProducerService := tt.fields.orderProducerService()

			uc := usecase.NewUseCase(orderRepository, nil, paymentClient, orderProducerService, nil)

			result, err := uc.PayOrder(ctx, testUUID, order_v1.PaymentMethodPAYMENTMETHODCARD)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, transactionUUID, result)
		})
	}
}
