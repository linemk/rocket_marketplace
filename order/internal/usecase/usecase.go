package usecase

import (
	"context"

	"github.com/google/uuid"

	inventoryClient "github.com/linemk/rocket-shop/order/internal/client/grpc/inventory/v1"
	paymentClient "github.com/linemk/rocket-shop/order/internal/client/grpc/payment/v1"
	"github.com/linemk/rocket-shop/order/internal/entyties/models"
	"github.com/linemk/rocket-shop/order/internal/metrics"
	"github.com/linemk/rocket-shop/order/internal/repository"
	"github.com/linemk/rocket-shop/order/internal/service"
	order_v1 "github.com/linemk/rocket-shop/shared/pkg/openapi/order/v1"
)

type OrderUseCase interface {
	CreateOrder(ctx context.Context, info OrderInfo) (string, error)
	GetOrder(ctx context.Context, uuid string) (models.Order, error)
	PayOrder(ctx context.Context, uuid string, paymentMethod order_v1.PaymentMethod) (string, error)
	CancelOrder(ctx context.Context, uuid string) error
}

type OrderInfo struct {
	UserID        string
	PartUUIDs     []uuid.UUID
	PaymentMethod order_v1.PaymentMethod
}

var _ OrderUseCase = (*useCase)(nil)

type useCase struct {
	orderRepository      repository.OrderRepository
	inventoryClient      inventoryClient.InventoryClient
	paymentClient        paymentClient.PaymentClient
	orderProducerService service.OrderProducerService
	metrics              *metrics.OrderMetrics
}

func NewUseCase(
	orderRepository repository.OrderRepository,
	inventoryClient inventoryClient.InventoryClient,
	paymentClient paymentClient.PaymentClient,
	orderProducerService service.OrderProducerService,
	metrics *metrics.OrderMetrics,
) OrderUseCase {
	return &useCase{
		orderRepository:      orderRepository,
		inventoryClient:      inventoryClient,
		paymentClient:        paymentClient,
		orderProducerService: orderProducerService,
		metrics:              metrics,
	}
}
