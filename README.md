# stdtask

Project Management System (ClickUp × Linear). Спека — [`PMS_Specification.md`](PMS_Specification.md);
проработанная документация — [`docs/`](docs/00-overview.md); план M0 — [`docs/m0-plan.md`](docs/m0-plan.md).

## Стек

- **Backend**: Go (chi, bun, ConnectRPC, coder/websocket, River), PostgreSQL + Redis.
- **Frontend**: Nuxt 3 + Nuxt UI + Pinia (local-first, optimistic UI, WS delta-sync).

## Локальный запуск (M0, Срез 1)

```bash
make up        # Postgres + Redis в Docker
make migrate   # применить миграции (ждёт готовности БД)
make run       # запустить сервер на :8080
# либо одной командой:
make dev
```

Проверка:
```bash
curl localhost:8080/health   # ok
curl localhost:8080/readyz   # ready (db + redis доступны)
```

## Структура

```
backend/   Go-сервис (cmd/, internal/, миграции)
frontend/  Nuxt-приложение (Срез 6)
proto/     контракты ConnectRPC (Срез 2)
docs/      документация и решения (ADR)
```
