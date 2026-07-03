.PHONY: client-dev server-run server-dev dev migrate-up migrate-down

client-dev:
	cd client && npm run dev

server-run:
	cd server && go run .

server-dev:
	@command -v air >/dev/null 2>&1 || { echo "air is not installed. Run: go install github.com/air-verse/air@latest"; exit 1; }
	cd server && air

migrate-up:
	cd server && make migrate-up

migrate-down:
	cd server && make migrate-down

dev:
	@command -v air >/dev/null 2>&1 || { echo "air is not installed. Run: go install github.com/air-verse/air@latest"; exit 1; }
	@trap 'kill 0' INT TERM EXIT; \
		(cd client && npm run dev) & \
		(cd server && air) & \
		wait
