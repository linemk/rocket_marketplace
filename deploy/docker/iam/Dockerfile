# Этап сборки
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Копируем все модули
COPY iam ./iam
COPY platform ./platform
COPY shared ./shared

# Собираем приложение без go.work (используем модули напрямую)
WORKDIR /app/iam
RUN go mod tidy
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /iam-service ./cmd/main.go

# Финальный этап
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копируем бинарник из builder
COPY --from=builder /iam-service .

# Копируем миграции
COPY --from=builder /app/iam/migrations ./migrations

# Указываем команду запуска
CMD ["./iam-service"]
