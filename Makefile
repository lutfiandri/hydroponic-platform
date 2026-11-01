.PHONY: up down logs admin migrate

up:
	mkdir -p pocketbase mqtt/data
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

admin:
	mkdir -p pocketbase
	docker compose stop pocketbase
	docker compose run --rm --no-deps --entrypoint sh pocketbase -ec \
		'exec pocketbase superuser upsert "$$POCKETBASE_ADMIN_EMAIL" "$$POCKETBASE_ADMIN_PASSWORD" \
		--dir=/pocketbase/data --migrationsDir=/tmp/pocketbase-migrations \
		--encryptionEnv=POCKETBASE_ENCRYPTION_KEY --automigrate=false'

migrate:
	mkdir -p pocketbase
	docker compose stop pocketbase
	docker compose run --rm --no-deps --entrypoint pocketbase pocketbase \
		migrate up --dir=/pocketbase/data --migrationsDir=/migrations \
		--encryptionEnv=POCKETBASE_ENCRYPTION_KEY --automigrate=false
