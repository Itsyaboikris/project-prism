.PHONY: run dev server-dev

run:
	go run .

dev:
	@command -v air >/dev/null 2>&1 || { echo "air is not installed. Run: go install github.com/air-verse/air@latest"; exit 1; }
	air

server-dev: dev
