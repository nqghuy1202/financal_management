# FinTrack — Hệ thống quản lý tài chính cá nhân

Ứng dụng quản lý tài chính cá nhân xây bằng Go, theo kiến trúc event-driven
với Kafka: ghi nhận giao dịch thu chi, theo dõi ngân sách theo danh mục,
đặt mục tiêu tiết kiệm và xem báo cáo chi tiêu.

> **Trạng thái:** đang phát triển. Xem [lộ trình chi tiết](docs/ROADMAP.md).
> Phase 0 (nền móng) đã hoàn thành.

## Tech stack

| Lớp | Công nghệ |
|---|---|
| Backend | Go 1.26, Gin |
| Database | PostgreSQL 17 (pgx), sqlc, goose |
| Cache / session | Redis 7 |
| Message broker | Apache Kafka (KRaft mode) |
| Frontend | Next.js 15, TypeScript, Tailwind, shadcn/ui |
| Observability | zap (JSON log), Prometheus, Grafana, OpenTelemetry |
| Hạ tầng | Docker Compose, GitHub Actions |

## Kiến trúc

```
Next.js ──REST/SSE──> cmd/api ──> PostgreSQL (+ bảng outbox)
                                       │
                                       ▼
                              cmd/outbox-relay ──> KAFKA
                                                     ├──> cmd/analytics-worker ──> read-model
                                                     └──> cmd/notification-worker ──> email + SSE
```

Hệ thống gồm một API service và các worker chạy độc lập, giao tiếp qua Kafka.
Mọi event đều được ghi vào bảng `outbox_events` trong cùng transaction với
dữ liệu nghiệp vụ, sau đó `outbox-relay` mới đẩy sang Kafka — nhờ vậy không
bao giờ xảy ra trường hợp ghi được database nhưng mất event, hoặc ngược lại.

## Yêu cầu môi trường

- [Go 1.26+](https://go.dev/dl/)
- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (bắt buộc,
  dùng để chạy PostgreSQL / Redis / Kafka)
- [Make](https://gnuwin32.sourceforge.net/packages/make.htm) (Windows)

## Chạy dự án

```bash
# 1. Tạo file cấu hình môi trường
cp .env.example .env

# 2. Khởi động hạ tầng (postgres, redis, kafka, kafka-ui, mailhog)
make up

# 3. Chạy API server
make run
```

Kiểm tra:

```bash
curl http://localhost:8080/healthz   # tiến trình còn sống
curl http://localhost:8080/readyz    # postgres + redis đã sẵn sàng
```

### Các giao diện phụ trợ

| Dịch vụ | Địa chỉ |
|---|---|
| API | http://localhost:8080 |
| Kafka UI | http://localhost:8081 |
| Mailhog (xem email) | http://localhost:8025 |
| PostgreSQL | localhost:**5433** |
| Redis | localhost:6379 |
| Kafka | localhost:9092 |

## Xử lý sự cố

**PostgreSQL dùng cổng 5433 chứ không phải 5432.** Nếu máy đã cài sẵn
PostgreSQL chạy như một dịch vụ Windows, nó giữ cổng 5432 trên IPv4 còn
Docker chỉ chiếm được IPv6. Ứng dụng khi đó kết nối trúng PostgreSQL của
máy thay vì container và báo `password authentication failed`. Dùng cổng
riêng để hai bên không tranh nhau. Đổi cổng thì phải sửa đồng thời
`POSTGRES_PORT` trong `.env` và `postgres.port` trong `config/local.yaml`.

**Docker Desktop cần được mở trước khi chạy `make up`** và mất khoảng 2–4
phút để daemon sẵn sàng. Nếu `docker ps` báo `cannot find the file
specified` thì daemon chưa lên, chờ thêm chứ chưa phải hỏng.

## Các lệnh thường dùng

```bash
make help    # xem toàn bộ lệnh
make watch   # chạy API với live reload
make test    # unit test (không cần docker)
make itest   # integration test (cần docker)
make check   # fmt + vet + lint + test, chạy trước khi commit
make down    # dừng hạ tầng
```

## Cấu trúc thư mục

```
cmd/
  api/                  # HTTP API server
config/
  local.yaml            # cấu hình môi trường dev
  production.yaml       # cấu hình production (secret override bằng env)
global/                 # dependency dùng chung, khởi tạo một lần lúc boot
internal/
  controller/           # HTTP handler — chỉ nhận request, gọi service, trả response
  services/             # logic nghiệp vụ — không biết gì về HTTP
  repo/                 # truy cập dữ liệu
  model/                # struct thực thể nghiệp vụ
  routers/              # khai báo route theo module
  initialize/           # khởi tạo config, logger, postgres, redis, router
  middlewares/          # request id, log, recovery, error handler, CORS, rate limit
  pkg/
    logger/             # dựng logger zap
    response/           # envelope response, catalog mã lỗi, AppError
    setting/            # struct cấu hình + validate
migrations/             # migration SQL (goose)
deployments/            # cấu hình Prometheus, Grafana
docs/                   # roadmap, ADR
web/                    # frontend Next.js
```

## Cấu hình

Cấu hình được đọc từ `config/<APP_ENV>.yaml`, mặc định `APP_ENV=local`.

Mọi khoá đều override được bằng biến môi trường theo quy tắc `FM_` + đường
dẫn khoá viết hoa, dấu chấm đổi thành gạch dưới:

```
postgres.password  →  FM_POSTGRES_PASSWORD
server.port        →  FM_SERVER_PORT
jwt.secret         →  FM_JWT_SECRET
```

Cấu hình được validate ngay lúc khởi động — thiếu hoặc sai giá trị thì ứng
dụng dừng ngay với thông báo rõ ràng, thay vì chạy được rồi lỗi giữa chừng.

## Quyết định thiết kế đáng chú ý

- **Tiền lưu bằng số nguyên đơn vị nhỏ nhất** (`int64`), không dùng số thực —
  tránh sai số làm tròn của dấu phẩy động trong tính toán tài chính.
- **Transactional Outbox** — giải bài toán ghi đồng thời vào database và
  message broker mà không dùng distributed transaction.
- **Consumer idempotent** — Kafka đảm bảo at-least-once, nên consumer phải
  chịu được việc nhận lại cùng một message.
- **CQRS cho báo cáo** — bảng đọc riêng do worker dựng sẵn, tránh chạy truy
  vấn tổng hợp nặng trên bảng giao dịch mỗi lần mở dashboard.
- **Rate limit token bucket bằng Lua script trên Redis** — nguyên tử, không
  bị lỗi burst gấp đôi ở ranh giới cửa sổ như fixed window counter.

Chi tiết lý do từng lựa chọn xem trong [docs/ROADMAP.md](docs/ROADMAP.md).
