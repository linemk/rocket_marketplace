package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"

	"github.com/linemk/rocket-shop/order/internal/entyties/apperrors"

	"github.com/stretchr/testify/require"

	v1 "github.com/linemk/rocket-shop/order/internal/client/grpc/inventory/v1"
	"github.com/linemk/rocket-shop/order/internal/mocks"
	"github.com/linemk/rocket-shop/order/internal/usecase"
	"github.com/linemk/rocket-shop/platform/pkg/logger"
	order_v1 "github.com/linemk/rocket-shop/shared/pkg/openapi/order/v1"
)

func TestCreate(t *testing.T) {
	// Инициализируем no-op логгер для тестов
	logger.SetNopLogger()

	ctx := context.Background()
	partUUID1 := uuid.New()
	partUUID2 := uuid.New()

	type fields struct {
		inventoryClient func() *mocks.MockInventoryClient
		orderRepository func() *mocks.MockOrderRepository
		paymentClient   func() *mocks.MockPaymentClient
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "successfully create a part",
			fields: fields{
				inventoryClient: func() *mocks.MockInventoryClient {
					mockClient := mocks.NewMockInventoryClient(gomock.NewController(t)) // нужен для подсчета вызовов

					mockClient.EXPECT().GetPart(gomock.Any(), partUUID1).Return(
						v1.PartInfo{
							UUID:          partUUID1.String(),
							Name:          "Engine Part",
							Price:         100.0,
							StockQuantity: 5,
						}, nil,
					)

					mockClient.EXPECT().GetPart(gomock.Any(), partUUID2).Return(
						v1.PartInfo{
							UUID:          partUUID2.String(),
							Name:          "Wing Part",
							Price:         200.0,
							StockQuantity: 5,
						}, nil,
					)

					return mockClient
				},

				orderRepository: func() *mocks.MockOrderRepository {
					mockClient := mocks.NewMockOrderRepository(gomock.NewController(t))
					mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

					return mockClient
				},

				paymentClient: func() *mocks.MockPaymentClient {
					mockClient := mocks.NewMockPaymentClient(gomock.NewController(t))

					return mockClient
				},
			},
			wantErr: false,
		},
		{
			name: "error part not found in inventory client",
			fields: fields{
				inventoryClient: func() *mocks.MockInventoryClient {
					mockClient := mocks.NewMockInventoryClient(gomock.NewController(t))
					mockClient.EXPECT().GetPart(gomock.Any(), partUUID1).Return(v1.PartInfo{}, apperrors.ErrPartNotFound)

					return mockClient
				},

				orderRepository: func() *mocks.MockOrderRepository {
					mockClient := mocks.NewMockOrderRepository(gomock.NewController(t))

					return mockClient
				},
				paymentClient: func() *mocks.MockPaymentClient {
					mockClient := mocks.NewMockPaymentClient(gomock.NewController(t))

					return mockClient
				},
			},
			wantErr: true,
		},
		{
			name: "error part out stocks in inventory client",
			fields: fields{
				inventoryClient: func() *mocks.MockInventoryClient {
					mockClient := mocks.NewMockInventoryClient(gomock.NewController(t))
					mockClient.EXPECT().GetPart(gomock.Any(), partUUID1).Return(v1.PartInfo{}, apperrors.ErrPartOutOfStock)

					return mockClient
				},

				orderRepository: func() *mocks.MockOrderRepository {
					mockClient := mocks.NewMockOrderRepository(gomock.NewController(t))

					return mockClient
				},
				paymentClient: func() *mocks.MockPaymentClient {
					mockClient := mocks.NewMockPaymentClient(gomock.NewController(t))

					return mockClient
				},
			},
			wantErr: true,
		},
		{
			name: "repo error in create a part",
			fields: fields{
				inventoryClient: func() *mocks.MockInventoryClient {
					mockClient := mocks.NewMockInventoryClient(gomock.NewController(t)) // нужен для подсчета вызовов

					mockClient.EXPECT().GetPart(gomock.Any(), partUUID1).Return(
						v1.PartInfo{
							UUID:          partUUID1.String(),
							Name:          "Engine Part",
							Price:         100.0,
							StockQuantity: 5,
						}, nil,
					)

					mockClient.EXPECT().GetPart(gomock.Any(), partUUID2).Return(
						v1.PartInfo{
							UUID:          partUUID2.String(),
							Name:          "Wing Part",
							Price:         200.0,
							StockQuantity: 5,
						}, nil,
					)

					return mockClient
				},

				orderRepository: func() *mocks.MockOrderRepository {
					mockClient := mocks.NewMockOrderRepository(gomock.NewController(t))
					mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(fmt.Errorf("failed to create order"))

					return mockClient
				},

				paymentClient: func() *mocks.MockPaymentClient {
					mockClient := mocks.NewMockPaymentClient(gomock.NewController(t))

					return mockClient
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inventoryClient := tt.fields.inventoryClient()
			orderRepository := tt.fields.orderRepository()
			paymentClient := tt.fields.paymentClient()

			uc := usecase.NewUseCase(orderRepository, inventoryClient, paymentClient, nil, nil)

			orderInfo := usecase.OrderInfo{
				UserID:        "user-123",
				PartUUIDs:     []uuid.UUID{partUUID1, partUUID2},
				PaymentMethod: order_v1.PaymentMethodPAYMENTMETHODUNSPECIFIED,
			}

			orderUUID, err := uc.CreateOrder(ctx, orderInfo)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, orderUUID)
			_, err = uuid.Parse(orderUUID)
			require.NoError(t, err)
		})
	}
}
