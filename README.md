# Rocket Marketplace - Микросервисная платформа для продажи ракет 🚀

Проект представляет собой распределенную систему на Go с event-driven архитектурой на базе Apache Kafka.

## Оглавление

- [Установка](#установка)
- [Запуск системы](#запуск-системы)
- [Envoy API Gateway](#-envoy-api-gateway)
- [Observability & Мониторинг](#-observability--мониторинг)
- [Полный Flow: IAM + Order + Inventory](#-полный-flow-iam--order--inventory-sl-5-iam)
- [Тестовые сценарии](#тестовые-сценарии)
- [API Reference](#api-reference)
- [Архитектура](#архитектура)

---

## Установка

### Требования

- Go 1.24+
- Docker & Docker Compose
- Task CLI
- PostgreSQL
- MongoDB
- Apache Kafka

### Установка Task CLI

```bash
brew install go-task
```

---

## Запуск системы

### 1. Запуск инфраструктуры

```bash
# Создать Docker сеть
docker network create rocket-shop-network

# Запустить инфраструктуру (Kafka + PostgreSQL + MongoDB + Redis + Observability)
task infra:up

# Заполнить БД тестовыми данными
task db:seed
```

**Observability стек запускается автоматически и доступен по адресам:**
- **Kibana** (логи): http://localhost:5601
- **Grafana** (метрики): http://localhost:3000 (admin/admin)
- **Jaeger** (трейсы): http://localhost:16686
- **Prometheus**: http://localhost:9099

### 2. Запуск микросервисов

```bash
# Запустить все сервисы
task start:all

# Или запустить отдельные сервисы
task services:start:iam
task services:start:inventory
task services:start:payment
task services:start:order
task services:start:assembly
task services:start:notification
```

**Для включения отправки логов и трейсов в Observability стек:**
```bash
# Установить переменные окружения перед запуском сервисов
export OTLP_ENABLED=true
export OTLP_ENDPOINT=localhost:4317
export SERVICE_NAME=order-service  # имя сервиса

# Запустить сервис
task services:start:order
```

### 3. Остановка системы

```bash
# Остановить все сервисы (рекомендуется)
task stop:all

# Или остановить отдельные компоненты
task services:stop          # Остановить сервисы
task db:down               # Остановить базы данных
task observability:down    # Остановить Observability стек
```

---

## 🌐 Envoy API Gateway

Проект использует **Envoy Proxy** как единую точку входа (API Gateway) для всех HTTP/REST запросов.

### Архитектура

```
                     Клиент (HTTP/REST)
                            ↓
                    Envoy Gateway :8080
                            ↓
        ┌───────────────────┼───────────────────┐
        ↓                   ↓                   ↓
   IAM gRPC           Order HTTP          Inventory gRPC
  (аутентификация)    (REST API)      (gRPC→JSON transcoding)
```

### Основные возможности

1. **Единая точка входа** - все API запросы идут через `localhost:8080`
2. **gRPC-JSON Transcoding** - автоматическое преобразование REST → gRPC для Inventory Service
3. **Централизованная аутентификация** - External Authorization через IAM Service
4. **CORS** - настроенная поддержка кросс-доменных запросов
5. **Маршрутизация** - автоматическая балансировка между микросервисами

### Endpoints

| Путь | Сервис | Протокол | Аутентификация |
|------|--------|----------|----------------|
| `/healthz` | Envoy | HTTP | ❌ Нет |
| `/auth/register` | IAM | HTTP→gRPC | ❌ Нет |
| `/auth/login` | IAM | HTTP→gRPC | ❌ Нет |
| `/api/v1/orders` | Order | HTTP→HTTP | ✅ Да |
| `/api/v1/inventory/parts` | Inventory | HTTP→gRPC | ✅ Да |

### Аутентификация через Envoy

Envoy использует **External Authorization** фильтр для проверки сессий:

```bash
# Все защищенные endpoints требуют заголовок с Session UUID
curl http://localhost:8080/api/v1/orders \
  -H "X-Session-UUID: <your-session-uuid>"

# Или через Cookie
curl http://localhost:8080/api/v1/inventory/parts \
  -H "Cookie: X-Session-Uuid=<your-session-uuid>"
```

**Как это работает:**
1. Клиент отправляет запрос с `X-Session-UUID` заголовком
2. Envoy перехватывает запрос через `ext_authz` фильтр
3. Envoy вызывает IAM Service gRPC метод для проверки сессии
4. Если сессия валидна - запрос проксируется к целевому сервису
5. Если сессия невалидна - возвращается `403 Forbidden`

### Запуск Envoy Gateway

```bash
# Запустить Envoy отдельно
task envoy:up

# Остановить
task envoy:down

# Перезапустить
task envoy:restart

# Посмотреть логи
task envoy:logs

# Протестировать Gateway (полный flow с регистрацией/логином)
task envoy:test
```

### Порты

- **8080** - основной API Gateway (все HTTP/REST запросы)
- **8081** - Envoy Admin UI (статистика, метрики, конфигурация)

### Admin UI

Envoy предоставляет административный интерфейс для мониторинга:

```bash
# Открыть Admin UI
open http://localhost:8081

# Полезные endpoints:
# - /stats - метрики и статистика
# - /config_dump - текущая конфигурация
# - /clusters - статус upstream кластеров
# - /listeners - активные listeners
```

### Конфигурация

Конфигурация Envoy находится в [deploy/compose/envoy/envoy.yaml](deploy/compose/envoy/envoy.yaml)

**Ключевые настройки:**
- **gRPC-JSON Transcoding**: использует proto descriptors из `/etc/envoy/combined_descriptor.pb`
- **External Authorization**: интеграция с IAM Service для проверки сессий
- **HTTP/2 для gRPC**: автоматическая конфигурация для gRPC upstream
- **Circuit Breakers**: защита от перегрузки IAM Service

---

## 📊 Observability & Мониторинг

Проект использует OpenTelemetry для сбора логов, метрик и трейсов.

### Архитектура

```
                    Микросервисы
                         ↓
        ┌────────────────┼────────────────┐
        ↓                ↓                ↓
   OTLP (4317)      OTLP (4317)   Prometheus HTTP
   (логи)           (трейсы)       (метрики)
        ↓                ↓                ↓
   OTel Collector   OTel Collector   Prometheus
        ↓                ↓                ↓
   Elasticsearch       Jaeger          Grafana
        ↓                ↓
     Kibana         Jaeger UI
```

### Компоненты стека

| Компонент | Назначение | Порт | URL |
|-----------|-----------|------|-----|
| **OpenTelemetry Collector** | Сбор и маршрутизация телеметрии | 4317 (gRPC) | - |
| **Elasticsearch** | Хранилище логов | 9200 | http://localhost:9200 |
| **Kibana** | UI для просмотра логов | 5601 | http://localhost:5601 |
| **Prometheus** | Сбор и хранение метрик | 9099 | http://localhost:9099 |
| **Grafana** | Визуализация метрик | 3000 | http://localhost:3000 |
| **Jaeger** | Визуализация распределённых трейсов | 16686 | http://localhost:16686 |

### Переменные окружения для сервисов

Добавьте в `.env` файл каждого сервиса:

```bash
# Включить отправку телеметрии в OpenTelemetry
OTLP_ENABLED=true

# Адрес OpenTelemetry Collector
OTLP_ENDPOINT=localhost:4317

# Имя сервиса (используется для фильтрации логов/трейсов)
SERVICE_NAME=order-service
```

### Как работает

1. **Логи и трейсы**: сервисы отправляют в **OpenTelemetry Collector** через gRPC (порт 4317)
   - **Логи** → OTel Collector → Elasticsearch → просмотр в Kibana
   - **Трейсы** → OTel Collector → Jaeger → визуализация распределённых вызовов

2. **Метрики**: Prometheus напрямую скрапит HTTP endpoints сервисов (порты 9090-9095)
   - Order Service: 9090
   - Assembly Service: 9091
   - Inventory Service: 9092
   - Payment Service: 9093
   - Notification Service: 9094
   - IAM Service: 9095
   - **Метрики** → Prometheus → визуализация в Grafana

### Команды управления

```bash
# Запустить Observability стек отдельно
task observability:up

# Остановить
task observability:down

# Перезапустить
task observability:restart

# Посмотреть логи стека
task observability:logs

# Открыть Kibana в браузере
task logs:kibana
```

---

## 🔐 Полный Flow: IAM + Order + Inventory (SL-5-IAM)

### Тестирование всего процесса: Register → Login → Create Order

```bash
# 1. Убедитесь, что система запущена
task start:all

# 2. Запустите полный тест
task test-api
```

**Что происходит в полном flow:**

1. **Register пользователя** (IAM Service)
   - Создает пользователя в PostgreSQL
   - Хэширует пароль (bcrypt)
   - Возвращает UUID пользователя

2. **Login** (IAM Service)
   - Проверяет учетные данные
   - Создает сессию в Redis (TTL = 24 часа)
   - Добавляет сессию в множество пользователя
   - Возвращает Session UUID

3. **List Parts** (Inventory Service)
   - Получает Session UUID из gRPC metadata (`session-uuid` заголовок)
   - Валидирует сессию через IAM Interceptor
   - Возвращает список доступных частей из MongoDB

4. **Create Order** (Order Service)
   - Получает Session UUID из HTTP заголовка (`X-Session-UUID`)
   - Передает Session UUID в gRPC metadata при вызове Inventory
   - Проверяет наличие деталей в Inventory
   - Создает заказ в PostgreSQL
   - Возвращает Order UUID

5. **Pay Order** (Order Service)
   - Обновляет статус заказа на `PAID`
   - Отправляет событие `OrderPaid` в Kafka
   - Assembly сервис начинает сборку корабля

6. **Cancel Order** (Order Service)
   - Отменяет заказ со статусом `PENDING_PAYMENT`
   - Обновляет статус на `CANCELLED`

### Redis & Session Management

#### Проверка сессий в Redis

```bash
# Подключиться к Redis
redis-cli

# Посмотреть все сессии
KEYS "session:*"

# Посмотреть данные конкретной сессии
GET "session:<SESSION_UUID>"

# Посмотреть все сессии пользователя
SMEMBERS "user_sessions:<USER_UUID>"

# Проверить TTL сессии
TTL "session:<SESSION_UUID>"

# Очистить сессии (для тестирования)
DEL "session:<SESSION_UUID>"
FLUSHDB
```

### gRPC + Reflection

#### Проверка доступных сервисов

```bash
# Список всех сервисов на IAM
grpcurl -plaintext localhost:50053 list

# Методы AuthService
grpcurl -plaintext localhost:50053 list auth.v1.AuthService

# Методы UserService
grpcurl -plaintext localhost:50053 list user.v1.UserService

# Методы InventoryService
grpcurl -plaintext localhost:50051 list inventory.v1.InventoryService

# Структура сообщения
grpcurl -plaintext localhost:50053 describe auth.v1.LoginRequest
```

### Примеры тестирования

#### 1. Регистрация

```bash
grpcurl -plaintext \
  -d '{"login":"test-user","password":"secret123","email":"test@example.com","notification_methods":[]}' \
  localhost:50053 user.v1.UserService/Register
```

#### 2. Логин

```bash
grpcurl -plaintext \
  -d '{"login":"test-user","password":"secret123"}' \
  localhost:50053 auth.v1.AuthService/Login
```

**Ответ:**
```json
{
  "session_uuid": "5596703b-d136-408a-aca6-fc76a9e3481c"
}
```

#### 3. Список деталей (с Session UUID в metadata)

```bash
grpcurl -plaintext \
  -H "session-uuid: 5596703b-d136-408a-aca6-fc76a9e3481c" \
  localhost:50051 inventory.v1.InventoryService/ListParts
```

#### 4. Создание заказа (с Session UUID в HTTP заголовке)

```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -H "X-Session-UUID: 5596703b-d136-408a-aca6-fc76a9e3481c" \
  -d '{"user_uuid":"user-uuid-from-register","part_uuids":["part-uuid-1"]}'
```

#### 5. Проверка текущего пользователя

```bash
grpcurl -plaintext \
  -d '{"session_uuid":"5596703b-d136-408a-aca6-fc76a9e3481c"}' \
  localhost:50053 auth.v1.AuthService/Whoami
```

---

## Тестовые сценарии

### Сценарий 1: Создание заказа

#### HTTP Request (curl)

```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "user-123",
    "partUUIDs": [
      "00000000-0000-0000-0000-000000000001",
      "00000000-0000-0000-0000-000000000002"
    ]
  }'
```

#### HTTP Request (Postman)

```
POST http://localhost:8080/api/v1/orders
Headers:
  Content-Type: application/json

Body (JSON):
{
  "userId": "user-123",
  "partUUIDs": [
    "00000000-0000-0000-0000-000000000001",
    "00000000-0000-0000-0000-000000000002"
  ]
}
```

#### Ожидаемый Response

```json
{
  "orderUuid": "851bc3b0-a4c7-43d5-a557-33473b33747b"
}
```

**Статус:** `201 Created`

#### Что произойдет

**В базе данных (PostgreSQL):**
- Создается запись в таблице `orders` со статусом `PENDING_PAYMENT`
- Сохраняется информация: `user_id`, `total_price`, `created_at`

**В Kafka:**
- Пока ничего (события отправляются только при оплате)

**В Telegram:**
- Пока ничего

---

### Сценарий 2: Оплата заказа

#### HTTP Request (curl)

```bash
ORDER_UUID="851bc3b0-a4c7-43d5-a557-33473b33747b"  # UUID из предыдущего шага

curl -X POST "http://localhost:8080/api/v1/orders/${ORDER_UUID}/pay" \
  -H "Content-Type: application/json" \
  -d '{
    "paymentMethod": "PAYMENT_METHOD_CARD"
  }'
```

#### HTTP Request (Postman)

```
POST http://localhost:8080/api/v1/orders/{{ORDER_UUID}}/pay
Headers:
  Content-Type: application/json

Body (JSON):
{
  "paymentMethod": "PAYMENT_METHOD_CARD"
}
```

**Доступные методы оплаты:**
- `PAYMENT_METHOD_CARD` - банковская карта
- `PAYMENT_METHOD_CASH` - наличные
- `PAYMENT_METHOD_CRYPTO` - криптовалюта

#### Ожидаемый Response

```json
{
  "transactionUuid": "47d0b01e-ca98-432d-b4c1-9e1c1bdc3614"
}
```

**Статус:** `200 OK`

#### Что произойдет

**В базе данных (PostgreSQL):**
- Статус заказа обновляется: `PENDING_PAYMENT` → `PAID`
- Сохраняется `transaction_uuid` и `payment_method`
- Обновляется `updated_at`

**В Kafka:**
1. **Order Service** отправляет событие `OrderPaid` в топик `order-paid`:
   ```json
   {
     "eventUuid": "e1db47c2-3f35-4abf-83d9-d199f531c309",
     "orderUuid": "851bc3b0-a4c7-43d5-a557-33473b33747b",
     "userUuid": "user-123",
     "paymentMethod": "PAYMENT_METHOD_CARD",
     "transactionUuid": "47d0b01e-ca98-432d-b4c1-9e1c1bdc3614"
   }
   ```

2. **Assembly Service** читает событие из `order-paid` и начинает сборку корабля (2-10 секунд)

3. **Assembly Service** отправляет событие `ShipAssembled` в топик `ship-assembled`:
   ```json
   {
     "eventUuid": "0bf809b7-35c6-4d7f-95ca-b85249cfd6bd",
     "orderUuid": "851bc3b0-a4c7-43d5-a557-33473b33747b",
     "userUuid": "user-123",
     "buildTimeSec": "5"
   }
   ```

4. **Order Service** читает событие из `ship-assembled` и обновляет статус заказа: `PAID` → `ASSEMBLED`

**В Telegram (приходят 2 сообщения):**

1. **Сообщение об оплате** (сразу после оплаты):
   ```
   💳 Платеж успешно обработан

   Информация о платеже:
   • Заказ: 851bc3b0-a4c7-43d5-a557-33473b33747b
   • Пользователь: user-123
   • Метод оплаты: PAYMENT_METHOD_CARD
   • Транзакция: 47d0b01e-ca98-432d-b4c1-9e1c1bdc3614

   Спасибо за вашу покупку!
   ```

2. **Сообщение о сборке** (через 2-10 секунд):
   ```
   🚀 Ваш заказ собран!

   Информация о доставке:
   • Заказ: 851bc3b0-a4c7-43d5-a557-33473b33747b
   • Пользователь: user-123
   • Время сборки: 5 сек

   Ваш заказ готов к доставке!
   ```

---

### Сценарий 3: Получение информации о заказе

#### HTTP Request (curl)

```bash
ORDER_UUID="851bc3b0-a4c7-43d5-a557-33473b33747b"

curl -X GET "http://localhost:8080/api/v1/orders/${ORDER_UUID}"
```

#### HTTP Request (Postman)

```
GET http://localhost:8080/api/v1/orders/{{ORDER_UUID}}
```

#### Ожидаемый Response (после оплаты и сборки)

```json
{
  "uuid": "851bc3b0-a4c7-43d5-a557-33473b33747b",
  "userId": "user-123",
  "totalPrice": 150000.00,
  "status": "ASSEMBLED",
  "paymentMethod": "PAYMENT_METHOD_CARD",
  "transactionUuid": "47d0b01e-ca98-432d-b4c1-9e1c1bdc3614",
  "createdAt": "2025-11-18T17:58:10Z",
  "updatedAt": "2025-11-18T17:58:20Z"
}
```

**Статус:** `200 OK`

**Возможные статусы заказа:**
- `PENDING_PAYMENT` - ожидает оплаты
- `PAID` - оплачен, но еще не собран
- `ASSEMBLED` - собран и готов к доставке
- `CANCELLED` - отменен

---

### Сценарий 4: Отмена заказа

#### HTTP Request (curl)

```bash
ORDER_UUID="851bc3b0-a4c7-43d5-a557-33473b33747b"

curl -X DELETE "http://localhost:8080/api/v1/orders/${ORDER_UUID}"
```

#### HTTP Request (Postman)

```
DELETE http://localhost:8080/api/v1/orders/{{ORDER_UUID}}
```

#### Ожидаемый Response

**Статус:** `204 No Content`

#### Что произойдет

**В базе данных (PostgreSQL):**
- Статус заказа обновляется на `CANCELLED`
- Обновляется `updated_at`

**В Kafka:**
- Пока ничего (в будущем можно добавить событие `OrderCancelled`)

**В Telegram:**
- Пока ничего

**Примечание:** Отменить можно только заказ в статусе `PENDING_PAYMENT`. Оплаченные заказы отменить нельзя.

---

## API Reference

### Orders API

| Метод | Endpoint | Описание |
|-------|----------|----------|
| POST | `/api/v1/orders` | Создать новый заказ |
| GET | `/api/v1/orders/{uuid}` | Получить информацию о заказе |
| POST | `/api/v1/orders/{uuid}/pay` | Оплатить заказ |
| DELETE | `/api/v1/orders/{uuid}` | Отменить заказ |

### Коды ответов

| Код | Описание |
|-----|----------|
| 200 | OK - Запрос выполнен успешно |
| 201 | Created - Ресурс создан |
| 204 | No Content - Запрос выполнен, тело ответа пустое |
| 400 | Bad Request - Некорректный запрос |
| 404 | Not Found - Ресурс не найден |
| 409 | Conflict - Конфликт (например, заказ уже оплачен) |
| 500 | Internal Server Error - Внутренняя ошибка сервера |

---

## Архитектура

### Микросервисы

1. **Order Service** (HTTP API + Kafka Producer + Kafka Consumer)
   - Управление заказами (CRUD)
   - Отправка событий `OrderPaid` в Kafka
   - Прием событий `ShipAssembled` из Kafka
   - База данных: PostgreSQL

2. **Assembly Service** (Kafka Consumer + Kafka Producer)
   - Симуляция сборки кораблей
   - Прием событий `OrderPaid` из Kafka
   - Отправка событий `ShipAssembled` в Kafka

3. **Notification Service** (Kafka Consumer)
   - Отправка уведомлений в Telegram
   - Прием событий `OrderPaid` и `ShipAssembled` из Kafka

4. **Payment Service** (gRPC Server)
   - Обработка платежей
   - Генерация transaction UUID

5. **Inventory Service** (gRPC Server)
   - Управление складом запчастей
   - База данных: MongoDB

### Event Flow

```
HTTP API → Order Service → order-paid → Assembly Service
    → ship-assembled → [Notification Service, Order Service]
    → Telegram
```

### Kafka Topics

- `order-paid` - события оплаты заказов (3 партиции)
- `ship-assembled` - события сборки кораблей (3 партиции)

### Consumer Groups

- `assembly-service` - читает из `order-paid`
- `notification-service-paid` - читает из `order-paid`
- `notification-service-assembled` - читает из `ship-assembled`
- `order-service` - читает из `ship-assembled`

---

## CI/CD

Проект использует GitHub Actions для непрерывной интеграции и доставки. Основные workflow:

- **CI** (`.github/workflows/ci.yml`) - проверяет код при каждом push и pull request
  - Линтинг кода
  - Проверка безопасности
  - Выполняется автоматическое извлечение версий из Taskfile.yml

---

## Troubleshooting

### Kafka не запускается

```bash
# Проверить логи Kafka
docker logs kafka

# Пересоздать контейнер
docker-compose -f deploy/compose/core/docker-compose.yml down
docker-compose -f deploy/compose/core/docker-compose.yml up -d
```

### Сервисы не могут подключиться к Kafka

```bash
# Проверить что Kafka доступен
docker ps | grep kafka

# Проверить что network создана
docker network ls | grep rocket-shop-network

# Если нет - создать
docker network create rocket-shop-network
```

### Telegram уведомления не приходят

1. Проверить логи notification service: `tail -f /tmp/notification.log`
2. Проверить что TELEGRAM_BOT_TOKEN и TELEGRAM_BOT_CHAT_ID правильные
3. Проверить что бот добавлен в чат
