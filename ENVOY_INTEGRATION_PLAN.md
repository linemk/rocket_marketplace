# План интеграции Envoy Gateway с аутентификацией

## Текущее состояние

### ✅ Работает
- Все сервисы запущены в Docker (IAM, Inventory, Order)
- Envoy Gateway запущен на порту 8080
- Envoy healthcheck работает (`/healthz`)
- Envoy ext_authz работает - блокирует неавторизованные запросы к:
  - `/api/v1/inventory/*` → 403 Forbidden
  - `/api/v1/orders` → 403 Forbidden
- HTTP→gRPC transcoding работает для Inventory Service
- Базы данных заполнены тестовыми данными

### ✅ Теперь работает
- HTTP→gRPC transcoding для `/auth/login` и `/auth/whoami`
- Envoy использует объединённый дескриптор (auth + inventory) для transcoding

---

## Задачи для полной интеграции

### 1. Добавить HTTP аннотации в auth.proto

**Файл:** `shared/proto/auth/v1/auth.proto`

Добавить google.api.http аннотации для методов:
- `Register` → `POST /auth/register`
- `Login` → `POST /auth/login`
- `Logout` → `POST /auth/logout`
- `Whoami` → `GET /auth/whoami`

**Пример:**
```protobuf
import "google/api/annotations.proto";

service AuthService {
  rpc Register(RegisterRequest) returns (RegisterResponse) {
    option (google.api.http) = {
      post: "/auth/register"
      body: "*"
    };
  }

  rpc Login(LoginRequest) returns (LoginResponse) {
    option (google.api.http) = {
      post: "/auth/login"
      body: "*"
    };
  }

  rpc Logout(LogoutRequest) returns (LogoutResponse) {
    option (google.api.http) = {
      post: "/auth/logout"
      body: "*"
    };
  }

  rpc Whoami(WhoamiRequest) returns (WhoamiResponse) {
    option (google.api.http) = {
      get: "/auth/whoami"
    };
  }
}
```

### 2. Сгенерировать auth descriptor

**Команда:**
```bash
task proto:build:auth
```

**Создать задачу в Taskfile.yml:**
```yaml
proto:build:auth:
  desc: Сборка proto дескриптора для Auth сервиса (для Envoy gRPC-JSON transcoder)
  deps: [ install-buf, proto:install-plugins ]
  dir: shared/proto
  cmds:
    - 'mkdir -p ../pkg/proto/auth/v1'
    - '{{.BUF}} build --path auth --as-file-descriptor-set --output "../pkg/proto/auth/v1/auth_descriptor.pb"'
```

### 3. Обновить Envoy конфигурацию

**Файл:** `deploy/compose/envoy/envoy.yaml`

Изменить маршрут `/auth/*` чтобы использовать grpc_json_transcoder:

```yaml
- match:
    prefix: "/auth/"
  route:
    cluster: iam_grpc_cluster
    timeout: 30s
  typed_per_filter_config:
    envoy.filters.http.ext_authz:
      "@type": type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthzPerRoute
      disabled: true
    envoy.filters.http.grpc_json_transcoder:
      "@type": type.googleapis.com/envoy.extensions.filters.http.grpc_json_transcoder.v3.GrpcJsonTranscoder
      proto_descriptor: "/etc/envoy/auth_descriptor.pb"
      services: ["auth.v1.AuthService"]
      print_options:
        add_whitespace: true
        always_print_primitive_fields: true
        always_print_enums_as_ints: false
        preserve_proto_field_names: false
```

### 4. Обновить docker-compose для Envoy

**Файл:** `deploy/compose/envoy/docker-compose.yml`

Добавить монтирование auth_descriptor.pb:

```yaml
volumes:
  - ./envoy.yaml:/etc/envoy/envoy.yaml:ro
  - ../../../shared/pkg/proto/inventory/v1/inventory_descriptor.pb:/etc/envoy/inventory_descriptor.pb:ro
  - ../../../shared/pkg/proto/auth/v1/auth_descriptor.pb:/etc/envoy/auth_descriptor.pb:ro
```

### 5. Перезапустить Envoy

```bash
docker-compose -f deploy/compose/envoy/docker-compose.yml down
docker-compose -f deploy/compose/envoy/docker-compose.yml up -d
```

---

## Тестирование после интеграции

### Полный флоу с аутентификацией через Envoy

```bash
# 1. Healthcheck
curl http://localhost:8080/healthz

# 2. Регистрация пользователя
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "login": "testuser",
    "password": "password123",
    "email": "test@example.com"
  }'

# 3. Логин
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "login": "testuser",
    "password": "password123"
  }' -c cookies.txt

# 4. Попытка доступа к Inventory БЕЗ аутентификации (должен вернуть 403)
curl http://localhost:8080/api/v1/inventory/parts

# 5. Доступ к Inventory С аутентификацией
curl -b cookies.txt http://localhost:8080/api/v1/inventory/parts

# 6. Создание заказа С аутентификацией
curl -b cookies.txt -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "user_uuid": "00000000-0000-0000-0000-000000000123",
    "part_uuids": [
      "78ddfbfd-697d-491c-9a7e-d9a36e44834a",
      "87a8637b-d35c-4bd1-a66d-9800f5f73561"
    ]
  }'

# 7. Оплата заказа
ORDER_UUID="<uuid из шага 6>"
curl -b cookies.txt -X POST "http://localhost:8080/api/v1/orders/${ORDER_UUID}/pay" \
  -H "Content-Type: application/json" \
  -d '{
    "paymentMethod": "PAYMENT_METHOD_CARD"
  }'

# 8. Проверка статуса заказа
curl -b cookies.txt "http://localhost:8080/api/v1/orders/${ORDER_UUID}"

# 9. Logout
curl -b cookies.txt -X POST http://localhost:8080/auth/logout

# 10. Проверка что после logout доступ закрыт
curl -b cookies.txt http://localhost:8080/api/v1/inventory/parts
```

---

## Обновление README.md

### Добавить раздел "Аутентификация через Envoy Gateway"

```markdown
## Аутентификация через Envoy Gateway

Все запросы к API проходят через Envoy Gateway с внешней авторизацией (ext_authz) через IAM сервис.

### Endpoints без аутентификации
- `GET /healthz` - healthcheck
- `POST /auth/register` - регистрация пользователя
- `POST /auth/login` - авторизация

### Endpoints с обязательной аутентификацией
- `GET /api/v1/inventory/parts` - список запчастей
- `GET /api/v1/inventory/parts/{uuid}` - информация о запчасти
- `POST /api/v1/orders` - создание заказа
- `POST /api/v1/orders/{uuid}/pay` - оплата заказа
- `GET /api/v1/orders/{uuid}` - информация о заказе

### Пример использования с аутентификацией

#### 1. Регистрация
\`\`\`bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "login": "user123",
    "password": "securepass",
    "email": "user@example.com"
  }'
\`\`\`

#### 2. Логин (сохраняем cookies)
\`\`\`bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "login": "user123",
    "password": "securepass"
  }' -c cookies.txt
\`\`\`

Ответ:
\`\`\`json
{
  "session_uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "user_uuid": "12345678-1234-1234-1234-123456789012"
}
\`\`\`

#### 3. Использование API с cookies
\`\`\`bash
# Получить список запчастей
curl -b cookies.txt http://localhost:8080/api/v1/inventory/parts

# Создать заказ
curl -b cookies.txt -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "user_uuid": "12345678-1234-1234-1234-123456789012",
    "part_uuids": ["part-uuid-1", "part-uuid-2"]
  }'
\`\`\`

#### 4. Logout
\`\`\`bash
curl -b cookies.txt -X POST http://localhost:8080/auth/logout
\`\`\`
```

---

## Обновление Taskfile.yml

### Добавить задачи для тестирования

```yaml
# ===============================
# Тестирование
# ===============================

test:
  desc: "Запустить все unit тесты"
  cmds:
    - echo "🧪 Запуск unit тестов..."
    - go test -v -race -coverprofile=coverage.out ./...
    - echo "✅ Unit тесты завершены"

test:integration:
  desc: "Запустить интеграционные тесты"
  deps: [ infra:up ]
  cmds:
    - echo "🧪 Запуск интеграционных тестов..."
    - cd inventory && go test -v -tags=integration ./tests/integration/...
    - echo "✅ Интеграционные тесты завершены"

test:e2e:envoy:
  desc: "E2E тест Envoy Gateway с аутентификацией"
  deps: [ start:all ]
  cmds:
    - echo "🧪 Запуск E2E теста Envoy Gateway..."
    - |
      # Healthcheck
      echo "1. Проверка healthcheck..."
      curl -f http://localhost:8080/healthz || exit 1

      # Регистрация
      echo "2. Регистрация пользователя..."
      curl -f -X POST http://localhost:8080/auth/register \
        -H "Content-Type: application/json" \
        -d '{"login":"e2etest","password":"test123","email":"e2e@test.com"}' || exit 1

      # Логин
      echo "3. Логин..."
      curl -f -X POST http://localhost:8080/auth/login \
        -H "Content-Type: application/json" \
        -d '{"login":"e2etest","password":"test123"}' \
        -c /tmp/cookies.txt || exit 1

      # Проверка доступа без auth (должен вернуть 403)
      echo "4. Проверка блокировки без auth..."
      if curl -f http://localhost:8080/api/v1/inventory/parts 2>/dev/null; then
        echo "❌ Ошибка: доступ без аутентификации должен быть заблокирован"
        exit 1
      fi

      # Проверка доступа с auth
      echo "5. Проверка доступа с auth..."
      curl -f -b /tmp/cookies.txt http://localhost:8080/api/v1/inventory/parts || exit 1

      echo "✅ E2E тест успешно завершен"

lint:
  desc: "Запустить линтер (golangci-lint)"
  cmds:
    - echo "🔍 Запуск линтера..."
    - golangci-lint run ./...
    - echo "✅ Линтер завершен"

lint:fix:
  desc: "Автоматически исправить проблемы линтера"
  cmds:
    - echo "🔧 Исправление проблем линтера..."
    - golangci-lint run --fix ./...
    - echo "✅ Проблемы исправлены"

test:all:
  desc: "Запустить все тесты (unit + integration + lint)"
  cmds:
    - task: lint
    - task: test
    - task: test:integration
    - echo "🎉 Все тесты успешно завершены!"

ci:
  desc: "CI pipeline (lint + tests)"
  cmds:
    - task: lint
    - task: test
    - task: test:integration
    - task: test:e2e:envoy
```

### Обновить существующие задачи

```yaml
start:all:
  desc: "🚀 Запустить всю систему (Kafka + БД + все сервисы + Envoy)"
  deps: [ infra:up ]
  cmds:
    - task: db:seed
    - sleep 2
    - task: services:start:all
    - |
      echo ""
      echo "🎉 Вся система запущена!"
      echo "📋 Проверить статус:"
      echo "   - Docker контейнеры: docker ps"
      echo "   - Envoy Gateway: http://localhost:8080/healthz"
      echo "   - Envoy Admin: http://localhost:8081"
      echo "   - Логи сервисов: task services:logs:all"
      echo ""
      echo "📝 Тестирование:"
      echo "   - E2E тест: task test:e2e:envoy"
      echo "   - Unit тесты: task test"
      echo "   - Интеграционные тесты: task test:integration"
```

---

## Команды для быстрого старта

```bash
# Полный запуск системы
task start:all

# Запустить все тесты
task test:all

# E2E тест с Envoy
task test:e2e:envoy

# Остановить систему
task stop:all
```

---

## Архитектура с Envoy Gateway

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client (HTTP/JSON)                       │
└────────────────────────────────┬────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Envoy Gateway (:8080)                         │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Filters:                                                  │   │
│  │  1. CORS                                                  │   │
│  │  2. ext_authz → IAM Service (session validation)        │   │
│  │  3. grpc_json_transcoder (HTTP↔gRPC)                    │   │
│  │  4. router                                               │   │
│  └──────────────────────────────────────────────────────────┘   │
└───┬─────────────────────┬─────────────────────┬────────────────┘
    │                     │                     │
    │ gRPC                │ gRPC                │ HTTP
    ▼                     ▼                     ▼
┌────────────┐      ┌──────────────┐      ┌─────────────┐
│   IAM      │      │  Inventory   │      │    Order    │
│  Service   │      │   Service    │      │   Service   │
│  (:50053)  │      │   (:50051)   │      │   (:8080)   │
└─────┬──────┘      └──────┬───────┘      └──────┬──────┘
      │                    │                     │
      ▼                    ▼                     ▼
┌──────────┐        ┌──────────┐         ┌──────────┐
│PostgreSQL│        │ MongoDB  │         │PostgreSQL│
│  +Redis  │        │          │         │          │
└──────────┘        └──────────┘         └──────────┘
```

---

## Статус выполнения

- [x] Создать Dockerfiles для сервисов
- [x] Настроить docker-compose для всех сервисов
- [x] Создать базовую конфигурацию Envoy
- [x] Настроить ext_authz для IAM
- [x] Настроить HTTP→gRPC transcoding для Inventory
- [x] Интегрировать Envoy в Taskfile
- [x] Добавить HTTP аннотации в auth.proto
- [x] Сгенерировать auth_descriptor.pb
- [x] Создать объединённый дескриптор (auth + inventory)
- [x] Настроить HTTP→gRPC transcoding для Auth
- [x] Протестировать полный флоу аутентификации через Envoy
- [ ] Обновить README с примерами аутентификации
- [ ] Добавить задачи тестирования в Taskfile
- [ ] Создать E2E тесты
