# Этап сборки
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Копируем все модули
COPY order ./order
COPY platform ./platform
COPY shared ./shared

# Собираем приложение без go.work (используем модули напрямую)
WORKDIR /app/order
RUN go mod tidy
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /order-service ./cmd/main.go

# Финальный этап
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копируем бинарник из builder
COPY --from=builder /order-service .

# Копируем миграции
COPY --from=builder /app/order/migrations ./migrations

# Указываем команду запуска
CMD ["./order-service"]
