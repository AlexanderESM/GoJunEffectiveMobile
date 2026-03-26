# Subscriptions Service

REST-сервис для агрегации данных об онлайн подписках пользователей.

## Запуск

```bash
docker compose up --build
```

Сервис будет доступен на `http://localhost:8080`.  
Swagger-документация: `docs/swagger.yaml` (открыть через [editor.swagger.io](https://editor.swagger.io)).

## Конфигурация

Все параметры задаются в `.env`:

| Переменная     | Описание                  | По умолчанию                                                        |
|----------------|---------------------------|---------------------------------------------------------------------|
| `DATABASE_URL` | DSN подключения к Postgres | `postgres://postgres:postgres@db:5432/subscriptions?sslmode=disable` |
| `SERVER_ADDR`  | Адрес HTTP-сервера         | `:8080`                                                             |

## API

Все даты передаются в формате `MM-YYYY`.

### CRUDL подписок

| Метод    | Путь                    | Описание                  |
|----------|-------------------------|---------------------------|
| `POST`   | `/subscriptions`        | Создать подписку          |
| `GET`    | `/subscriptions`        | Список (фильтр `?user_id=`) |
| `GET`    | `/subscriptions/{id}`   | Получить по ID            |
| `PUT`    | `/subscriptions/{id}`   | Обновить подписку         |
| `DELETE` | `/subscriptions/{id}`   | Удалить подписку          |

### Подсчёт суммарной стоимости

```
GET /subscriptions/total?user_id=&service_name=&from=MM-YYYY&to=MM-YYYY
```

Все параметры опциональны.

### Пример создания подписки

```bash
curl -X POST http://localhost:8080/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "service_name": "Yandex Plus",
    "price": 299,
    "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
    "start_date": "07-2025"
  }'
```

### Пример подсчёта стоимости

```bash
curl "http://localhost:8080/subscriptions/total?user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba&from=01-2025&to=12-2025"
```

## Миграции

Миграция `migrations/001_init.up.sql` применяется автоматически при первом запуске PostgreSQL через `docker-entrypoint-initdb.d`.

Для ручного применения/отката:
```bash
psql $DATABASE_URL -f migrations/001_init.up.sql
psql $DATABASE_URL -f migrations/001_init.down.sql
```
