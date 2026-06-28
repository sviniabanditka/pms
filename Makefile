.PHONY: up down logs migrate run test proto s3-init

# Кодген proto → Go (backend/gen). Требует buf + protoc-gen-go + protoc-gen-connect-go на PATH.
proto:
	buf lint
	buf generate

# Инфраструктура (Postgres + Redis + Garage S3)
up:
	docker compose up -d

# Bootstrap Garage S3 (layout/ключ/бакет) — один раз после up
s3-init:
	sh docker/garage-init.sh

down:
	docker compose down

logs:
	docker compose logs -f

# Применить миграции (роль postgres). Требует поднятого compose.
migrate:
	cd backend && go run ./cmd/migrate

# Запустить сервер (роль pms_app).
run:
	cd backend && go run ./cmd/server

test:
	cd backend && go test ./...

# Полный локальный старт: инфра → ожидание готовности → миграции → сервер.
dev: up
	@echo "waiting for postgres..."
	@until docker compose exec -T postgres pg_isready -U postgres -d pms >/dev/null 2>&1; do sleep 1; done
	$(MAKE) migrate
	$(MAKE) run
