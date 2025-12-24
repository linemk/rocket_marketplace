package tests

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/linemk/rocket-shop/order/internal/entyties/apperrors"
	"github.com/linemk/rocket-shop/order/internal/entyties/models"
	"github.com/linemk/rocket-shop/order/internal/mocks"
	"github.com/linemk/rocket-shop/order/internal/usecase"
	order_v1 "github.com/linemk/rocket-shop/shared/pkg/openapi/order/v1"
)

func TestCancel(t *testing.T) {
	ctx := context.Background()
	testUUID := uuid.New()

	type fields struct {
		orderRepository func() *mocks.MockOrderRepository
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "successful cancel order",
			fields: fields{
				orderRepository: func() *mocks.MockOrderRepository {
					mockClient := mocks.NewMockOrderRepository(gomock.NewController(t))
					mockClient.EXPECT().Get(ctx, testUUID.String()).Return(models.Order{
						UUID:   testUUID.String(),
						Status: order_v1.OrderStatusPENDINGPAYMENT,
					}, nil)

					mockClient.EXPECT().Update(ctx, testUUID.String(), gomock.Any()).Return(nil)

					return mockClient
				},
			},
			wantErr: false,
		},
		{
			name: "error order not found",
			fields: fields{
				orderRepository: func() *mocks.MockOrderRepository {
					mockClient := mocks.NewMockOrderRepository(gomock.NewController(t))
					mockClient.EXPECT().Get(ctx, testUUID.String()).Return(models.Order{}, apperrors.ErrOrderNotFound)

					return mockClient
				},
			},
			wantErr: true,
		},
		{
			name: "error order already paid",
			fields: fields{
				orderRepository: func() *mocks.MockOrderRepository {
					mockClient := mocks.NewMockOrderRepository(gomock.NewController(t))
					mockClient.EXPECT().Get(ctx, testUUID.String()).Return(models.Order{
						UUID:   testUUID.String(),
						Status: order_v1.OrderStatusPAID,
					}, nil)

					return mockClient
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepository := tt.fields.orderRepository()

			uc := usecase.NewUseCase(orderRepository, nil, nil, nil, nil)

			err := uc.CancelOrder(ctx, testUUID.String())
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
