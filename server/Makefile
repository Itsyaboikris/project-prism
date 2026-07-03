.PHONY: run dev server-dev migrate-up migrate-down migrate-create

run:
	go run .

dev:
	@command -v air >/dev/null 2>&1 || { echo "air is not installed. Run: go install github.com/air-verse/air@latest"; exit 1; }
	air

server-dev: dev

migrate-up:
	@command -v migrate >/dev/null 2>&1 || { echo "migrate is not installed. Run: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"; exit 1; }
	migrate -path ./migrations -database "$(DATABASE_URL)" up

migrate-down:
	@command -v migrate >/dev/null 2>&1 || { echo "migrate is not installed. Run: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"; exit 1; }
	migrate -path ./migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@[ "$(name)" ] || { echo "Usage: make migrate-create name=your_migration_name"; exit 1; }
	migrate create -ext sql -dir ./migrations -seq $(name)

