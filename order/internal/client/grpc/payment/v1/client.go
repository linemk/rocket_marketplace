package v1

import (
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/linemk/rocket-shop/platform/pkg/tracing"
	payment_v1 "github.com/linemk/rocket-shop/shared/pkg/proto/payment/v1"
)

type Client struct {
	client payment_v1.PaymentServiceClient
	conn   *grpc.ClientConn
}

func NewClient(address string) (*Client, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// Добавляем client interceptor если tracer инициализирован
	if otel.GetTracerProvider() != noop.NewTracerProvider() {
		opts = append(opts, grpc.WithUnaryInterceptor(tracing.UnaryClientInterceptor()))
	}

	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PaymentService: %w", err)
	}

	client := payment_v1.NewPaymentServiceClient(conn)

	return &Client{
		client: client,
		conn:   conn,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// Проверяем, что Client реализует интерфейс PaymentClient
var _ PaymentClient = (*Client)(nil)
