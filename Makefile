.PHONY: dev-api dev-web test check build run gamedata

# Run the Go API on :8080 (the Svelte dev server proxies /api to it).
dev-api:
	cd backend && go run ./cmd/server -addr :8080

# Run the Svelte dev server on :5173.
dev-web:
	cd frontend && npm run dev

test:
	cd backend && go vet ./... && go test ./...

check: test
	cd frontend && npm run check

# Build the frontend and a single server binary that serves it.
build:
	cd frontend && npm ci && npm run build
	cd backend && go build -o ../bin/aion-scouter ./cmd/server

run: build
	./bin/aion-scouter -addr :8080 -static frontend/build

# Re-import skill tooltips and Daevanion boards from the Global client data.
gamedata:
	cd backend && go run ./cmd/gamedata -classes spiritmaster,chanter
