.DEFAULT_GOAL := help

BINARY_DIR := bin
API_BINARY := $(BINARY_DIR)/api.exe

# ---------------------------------------------------------------------
# Trợ giúp
# ---------------------------------------------------------------------
.PHONY: help
help: ## Hiển thị danh sách lệnh
	@powershell -NoProfile -Command "Select-String -Path Makefile -Pattern '^[a-zA-Z_-]+:.*?## .*$$' | ForEach-Object { $$p = $$_.Line -split ':.*?## '; '{0,-20} {1}' -f $$p[0], $$p[1] }"

# ---------------------------------------------------------------------
# Hạ tầng
# ---------------------------------------------------------------------
.PHONY: up
up: ## Khởi động toàn bộ hạ tầng (postgres, redis, kafka, kafka-ui, mailhog)
	docker compose up -d

.PHONY: down
down: ## Dừng hạ tầng, giữ lại dữ liệu
	docker compose down

.PHONY: clean-infra
clean-infra: ## Dừng hạ tầng và XOÁ toàn bộ dữ liệu trong volume
	docker compose down -v

.PHONY: logs
logs: ## Xem log của hạ tầng
	docker compose logs -f

.PHONY: ps
ps: ## Xem trạng thái các container
	docker compose ps

# ---------------------------------------------------------------------
# Database
# ---------------------------------------------------------------------
# goose đọc GOOSE_DRIVER / GOOSE_DBSTRING / GOOSE_MIGRATION_DIR từ .env
GOOSE := goose

.PHONY: tools
tools: ## Cài goose và sqlc
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

.PHONY: migrate-up
migrate-up: ## Chạy toàn bộ migration còn thiếu
	$(GOOSE) up

.PHONY: migrate-down
migrate-down: ## Lùi lại một migration
	$(GOOSE) down

.PHONY: migrate-status
migrate-status: ## Xem migration nào đã chạy
	$(GOOSE) status

.PHONY: migrate-reset
migrate-reset: ## Lùi toàn bộ migration (XOÁ HẾT dữ liệu)
	$(GOOSE) reset

.PHONY: migrate-create
migrate-create: ## Tạo migration mới: make migrate-create name=create_accounts
	$(GOOSE) -s create $(name) sql

.PHONY: sqlc
sqlc: ## Sinh lại code truy vấn từ SQL
	sqlc generate

.PHONY: sqlc-verify
sqlc-verify: ## Kiểm tra query có khớp schema không (không ghi file)
	sqlc vet
	sqlc diff

# ---------------------------------------------------------------------
# Ứng dụng
# ---------------------------------------------------------------------
.PHONY: build
build: ## Build API server
	go build -o $(API_BINARY) ./cmd/api

.PHONY: run
run: ## Chạy API server
	go run ./cmd/api

.PHONY: watch
watch: ## Chạy API server với live reload (air)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "if (Get-Command air -ErrorAction SilentlyContinue) { air } else { Write-Output 'Chua cai air, dang cai...'; go install github.com/air-verse/air@latest; air }"

# ---------------------------------------------------------------------
# Chất lượng code
# ---------------------------------------------------------------------
.PHONY: fmt
fmt: ## Format toàn bộ code
	go fmt ./...

.PHONY: vet
vet: ## Chạy go vet
	go vet ./...

.PHONY: lint
lint: ## Chạy golangci-lint
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "if (Get-Command golangci-lint -ErrorAction SilentlyContinue) { golangci-lint run ./... } else { Write-Output 'Chua cai golangci-lint. Cai bang: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest'; exit 1 }"

.PHONY: test
test: ## Chạy unit test (nhanh, không cần docker)
	go test -short -cover ./...

# -race cần cgo, mà cgo cần trình biên dịch C. Trên Windows thường không
# có sẵn gcc nên tách riêng: local dùng `test`, CI dùng `test-race`.
.PHONY: test-race
test-race: ## Chạy unit test kèm race detector (cần gcc/cgo)
	go test -short -race -cover ./...

.PHONY: itest
itest: ## Chạy integration test (cần docker chạy sẵn)
	go test -count=1 -tags=integration ./...

.PHONY: cover
cover: ## Chạy test và mở báo cáo coverage
	go test -short -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

.PHONY: check
check: fmt vet lint test ## Chạy toàn bộ kiểm tra trước khi commit

.PHONY: tidy
tidy: ## Dọn go.mod
	go mod tidy

.PHONY: clean
clean: ## Xoá file build
	@powershell -NoProfile -Command "if (Test-Path '$(BINARY_DIR)') { Remove-Item -Recurse -Force '$(BINARY_DIR)' }; if (Test-Path 'coverage.out') { Remove-Item -Force 'coverage.out' }"
