# Roadmap — Personal Finance Management (Go + Kafka + Next.js)

> Tài liệu này là kế hoạch thực hiện. Cập nhật trạng thái từng phase khi hoàn thành.

## 1. Quyết định đã chốt

| Hạng mục | Lựa chọn |
|---|---|
| Kiến trúc | Modular monolith (API) + 4 worker/binary riêng, giao tiếp qua Kafka |
| Backend | Go 1.26, Gin, sqlc, goose |
| Database | PostgreSQL 17 (chuyển từ MySQL), Redis 7 |
| Message broker | Apache Kafka (KRaft mode), client `segmentio/kafka-go` |
| Patterns | Transactional Outbox, Idempotent Consumer, Retry + DLQ, CQRS read-model |
| Frontend | Next.js 15 (App Router), TypeScript, Tailwind, shadcn/ui, TanStack Query, Recharts |
| Deploy | Docker Compose + GitHub Actions CI |
| Observability | zap (JSON logs), Prometheus + Grafana, OpenTelemetry + Jaeger |

---

## 2. Sản phẩm sẽ xây

**Tên đề xuất: FinTrack** — ứng dụng quản lý tài chính cá nhân.

### Trọng tâm sản phẩm

Người dùng nhập các khoản **thu và chi** hằng ngày. Hệ thống tổng hợp và
trả lại **bức tranh tài chính** theo tháng hoặc năm.

Phần báo cáo mới là giá trị cốt lõi; nhập liệu chỉ là đầu vào. Mọi quyết
định thiết kế đều phục vụ điều đó.

**Không có chức năng chuyển khoản.** Chuyển tiền giữa hai nguồn của chính
mình không làm đổi tổng thu hay tổng chi, nên không đổi bức tranh tài
chính — thêm vào chỉ khiến mọi truy vấn báo cáo phải nhớ loại trừ nó.

### Tính năng nghiệp vụ
1. **Auth** — đăng ký, đăng nhập, JWT access (15m) + refresh (7d), logout
2. **Nguồn tiền** — tiền mặt, ngân hàng, ví điện tử, thẻ tín dụng. Chỉ là
   nhãn để biết tiền đi qua đâu, **không theo dõi số dư**
3. **Danh mục** — thu/chi, phẳng, có icon và màu, seed sẵn 16 danh mục
4. **Giao dịch** — chỉ thu và chi, kèm số tiền, danh mục, ngày, ghi chú
5. **Báo cáo** — bốn góc nhìn, xem mục bên dưới
6. **Ngân sách** — hạn mức theo danh mục theo tháng, cảnh báo khi vượt
7. **Thông báo** — in-app realtime (SSE) + email

### Bốn báo cáo bắt buộc có
| Báo cáo | Trả lời câu hỏi |
|---|---|
| Chi tiêu theo danh mục | Tháng này tiền đi đâu? Danh mục nào chiếm nhiều nhất? |
| Dòng tiền theo thời gian | Thu và chi từng tháng trong năm biến động thế nào? |
| So sánh kỳ này với kỳ trước | Tháng này chi nhiều hơn tháng trước bao nhiêu? Do danh mục nào? |
| Tỷ lệ tiết kiệm và số dư ròng | Tháng này dư hay âm? Để dành được bao nhiêu phần trăm thu nhập? |

### Quyết định thiết kế quan trọng
- **Tiền lưu bằng `NUMERIC(19,4)`**, phía Go dùng `shopspring/decimal` bọc trong kiểu `model.Money` gắn liền số tiền với loại tiền tệ. **Tuyệt đối không dùng `float`.**
- **Không lưu số dư ở đâu cả.** Số dư ròng luôn được tính từ giao dịch, nên không bao giờ lệch với thực tế.
- **Mọi báo cáo tổng hợp theo `occurred_at`** (ngày tiền thực sự thu/chi), không theo `created_at`.
- **Mọi event ra Kafka đều đi qua Outbox**, không bao giờ publish trực tiếp trong handler.

---

## 3. Kiến trúc

```
                          ┌──────────────────────────┐
                          │   web/  Next.js 15       │
                          └────────────┬─────────────┘
                                       │ REST + JWT, SSE
                          ┌────────────▼─────────────┐
                          │  cmd/api   (Gin)         │
                          │  modules: auth, account, │
                          │  category, transaction,  │
                          │  budget, goal, report    │
                          └───┬──────────────────┬───┘
                              │                  │
                    ┌─────────▼──────┐   ┌───────▼──────┐
                    │  PostgreSQL    │   │    Redis     │
                    │  + outbox_     │   │ cache, rate  │
                    │    events      │   │ limit, token │
                    └─────────┬──────┘   └──────────────┘
                              │ poll & publish
                    ┌─────────▼────────┐
                    │ cmd/outbox-relay │
                    └─────────┬────────┘
                              │
                  ┌───────────▼─────────────┐
                  │         KAFKA           │
                  │  fin.transaction.v1     │
                  │  fin.budget.v1          │
                  │  fin.notification.v1    │
                  │  fin.notification.v1.dlq│
                  └──┬───────────────┬──────┘
                     │               │
        ┌────────────▼──────┐  ┌─────▼──────────────────┐
        │ cmd/analytics-    │  │ cmd/notification-      │
        │ worker  (CQRS)    │  │ worker (email + SSE)   │
        └────────────┬──────┘  └────────────────────────┘
                     │
        ┌────────────▼─────────┐      ┌──────────────────┐
        │ read-model tables    │      │ cmd/scheduler    │
        │ rm_monthly_summary   │      │ tong ket tuan,   │
        │ rm_category_spending │      │ tong ket thang   │
        └──────────────────────┘      └──────────────────┘
```

### 5 binary
| Binary | Vai trò |
|---|---|
| `cmd/api` | HTTP API, phục vụ frontend |
| `cmd/outbox-relay` | Đọc `outbox_events` → publish Kafka → đánh dấu đã gửi |
| `cmd/analytics-worker` | Consume transaction event → dựng read-model (CQRS) |
| `cmd/notification-worker` | Consume budget/notification event → email + push SSE |
| `cmd/scheduler` | Cron: gửi tổng kết tuần/tháng |

### Kafka topics
| Topic | Key | Events |
|---|---|---|
| `fin.transaction.v1` | `user_id` | `TransactionCreated`, `TransactionUpdated`, `TransactionDeleted` |
| `fin.budget.v1` | `user_id` | `BudgetThresholdReached`, `BudgetExceeded` |
| `fin.notification.v1` | `user_id` | `NotificationRequested` |
| `fin.notification.v1.dlq` | `user_id` | message xử lý thất bại sau N lần retry |

**Key = `user_id`** → đảm bảo mọi event của cùng một user vào cùng partition, giữ đúng thứ tự. Đây là điểm phải giải thích được khi phỏng vấn.

Consumer groups: `analytics-cg`, `notification-cg` — mỗi group đọc độc lập cùng một topic.

---

## 4. Những thứ tạo ấn tượng trong CV

Đây là danh sách "talking points" — mỗi mục là một câu hỏi phỏng vấn bạn sẽ trả lời được:

1. **Transactional Outbox** — giải bài toán dual-write: làm sao ghi DB và publish Kafka mà không mất event khi một trong hai fail
2. **Idempotent consumer** — bảng `processed_events` + upsert, xử lý at-least-once delivery của Kafka
3. **Retry + Dead Letter Queue** — exponential backoff, kèm CLI replay lại message từ DLQ
4. **CQRS read-model + event replay** — có lệnh dựng lại toàn bộ bảng thống kê bằng cách đọc lại Kafka từ offset 0
5. **Kiểu Money riêng** — số tiền luôn đi kèm loại tiền tệ, cộng hai loại tiền khác nhau là lỗi biên dịch được chặn, không bao giờ dùng float
6. **sqlc** — SQL thuần được sinh ra code Go type-safe, không ORM che giấu query
7. **Testcontainers** — integration test chạy Postgres + Kafka thật trong Docker (thư viện này đã có sẵn trong `go.mod` của bạn)
8. **Distributed tracing xuyên Kafka** — nhét trace context vào Kafka header, xem được một request đi từ HTTP → DB → Kafka → worker trên Jaeger
9. **Graceful shutdown + context propagation** ở cả API lẫn consumer

---

## 5. Lộ trình 8 phase

Ước lượng theo nhịp part-time ~10–15h/tuần. Mỗi phase kết thúc bằng một commit chạy được.

### Phase 0 — Dọn dẹp & nền móng · 3–4 ngày
**Vấn đề hiện tại cần xử lý:**
- `cmd/server/main.go` khai báo `package server` → không phải main package, không chạy được
- Trùng lặp: `internal/routers/router.go` giống hệt `internal/initialize/router.go`
- Code thừa từ go-blueprint không dùng: `internal/server/`, `internal/database/`
- Code học tập cần bỏ: `cmd/cli/`, `internal/pkg/tests/basic/`, các middleware `AA/BB/CC`
- `internal/pkg/setting/setting.go` — struct Config rỗng, viper unmarshal vào không có gì
- Middleware auth đang hardcode `token != "invalid-token"` (logic còn ngược)
- `log/log.txt` và `coverage.html` bị commit vào git → phải cho vào `.gitignore`

**Việc làm:**
- [ ] **Đổi tên module** `financal_management` → `github.com/<username>/fintrack` (tên hiện tại sai chính tả "financal", rất lộ trên CV)
- [ ] Xoá toàn bộ code trùng lặp / không dùng ở trên
- [ ] Chuẩn hoá layout thư mục: `cmd/ internal/ pkg/ migrations/ deployments/ docs/ web/`
- [ ] Config: struct đầy đủ + viper + override bằng env + validate lúc khởi động
- [ ] Logger zap production (JSON, level, caller, rotate bằng lumberjack)
- [ ] Chuẩn error: `AppError` type + error catalog + middleware recover
- [ ] Graceful shutdown
- [ ] `docker-compose.yml`: postgres, redis, kafka (KRaft), kafka-ui, mailhog
- [ ] Makefile + `golangci-lint`

**Xong khi:** `docker compose up -d && make run` → `GET /healthz` trả 200, log ra JSON.

---

### Phase 1 — Data layer & Auth · 1 tuần
- [ ] Thiết kế ERD, viết migration goose: `users, accounts, categories, transactions, budgets, goals, outbox_events, processed_events, notifications` + bảng read-model
- [ ] Cài sqlc, viết query đầu tiên, sinh code
- [ ] Repository interface + implementation, transaction manager (`WithTx`)
- [ ] Auth: `register / login / refresh / logout / me`
- [ ] Hash password bằng Argon2id
- [ ] JWT: access 15 phút, refresh 7 ngày có rotation, lưu refresh trong Redis để revoke được
- [ ] Middleware auth thật (thay cái hardcode), middleware rate-limit bằng Redis token bucket
- [ ] Unit test service + integration test repo bằng testcontainers

**Xong khi:** đăng ký → đăng nhập → gọi `/me` bằng token chạy đúng, có test tự động.

---

### Phase 2 — Core domain · 1.5 tuần
- [ ] Accounts CRUD, Categories CRUD + seed bộ danh mục mặc định tiếng Việt
- [ ] Transactions: create / update / delete / list với filter (khoảng ngày, ví, danh mục, loại, khoảng tiền) + cursor pagination
- [ ] Transfer giữa 2 ví trong cùng một DB transaction, có row-level lock
- [ ] Money type `int64` + currency, validate chặt bằng `validator/v10`
- [ ] Savings goals CRUD
- [ ] OpenAPI spec + Swagger UI

**Xong khi:** API tài chính đầy đủ, test coverage tầng domain > 70%.

---

### Phase 3 — Kafka & Outbox · 1 tuần ← **trái tim dự án**
- [ ] Wrapper Kafka producer/consumer (`segmentio/kafka-go`)
- [ ] Định nghĩa event schema có version trong `pkg/events` (JSON, kèm `event_id`, `occurred_at`, `trace_id`)
- [ ] Bảng `outbox_events` + ghi outbox **trong cùng transaction** với nghiệp vụ
- [ ] `cmd/outbox-relay`: poll → publish → mark sent, có retry + backoff, tránh race bằng `FOR UPDATE SKIP LOCKED`
- [ ] Idempotency: bảng `processed_events` (`consumer_group`, `event_id`)
- [ ] Consumer framework dùng chung: graceful shutdown, commit sau khi xử lý xong, retry, DLQ
- [ ] Integration test Kafka bằng testcontainers

**Xong khi:** tạo 1 giao dịch → thấy event xuất hiện trong kafka-ui, kill relay giữa chừng rồi bật lại → event vẫn được gửi đúng 1 lần về mặt hiệu ứng.

---

### Phase 4 — Analytics worker (CQRS) · 5–6 ngày
- [ ] Bảng read-model: `rm_monthly_summary`, `rm_category_spending`, `rm_daily_balance`
- [ ] `cmd/analytics-worker` consume transaction event → cập nhật read-model, xử lý idempotent và out-of-order
- [ ] Lệnh **replay**: đọc lại topic từ offset 0 để dựng lại toàn bộ read-model
- [ ] API `/v1/reports/*` đọc từ read-model + cache Redis

**Xong khi:** xoá sạch bảng read-model → chạy replay → số liệu dashboard khớp lại y nguyên.

---

### Phase 5 — Budget & Notification worker · 1 tuần
- [ ] Budget CRUD, tính `spent` theo period
- [ ] Consumer tính % ngân sách → phát `BudgetThresholdReached` / `BudgetExceeded`
- [ ] `cmd/notification-worker`: consume → lưu notification vào DB + gửi email qua Mailhog
- [ ] Retry exponential backoff + DLQ + CLI replay lại DLQ
- [ ] SSE endpoint `GET /v1/notifications/stream` để web nhận realtime
- [ ] `cmd/scheduler`: sinh giao dịch định kỳ, gửi tổng kết tuần

**Xong khi:** tạo giao dịch làm vượt ngân sách → nhận email trong Mailhog **và** toast hiện ngay trên web.

---

### Phase 6 — Frontend Next.js · 1–1.5 tuần

> Rút ngắn so với dự kiến ban đầu: tác giả đã thạo Next.js App Router,
> TypeScript và Tailwind v4 qua dự án `profolio`, nên không cần thời gian
> làm quen stack.

- [ ] Setup Next 15 App Router + Tailwind + shadcn/ui + TanStack Query
- [ ] API client với interceptor tự refresh token khi 401
- [ ] Auth flow + route protection bằng middleware
- [ ] **Dashboard**: KPI card (thu/chi/số dư/tiết kiệm), biểu đồ dòng tiền, donut chi tiêu theo danh mục
- [ ] **Transactions**: data table có filter, form dạng sheet, optimistic update
- [ ] **Accounts / Budgets** (progress bar theo mốc cảnh báo) **/ Goals**
- [ ] **Reports**: so sánh tháng, top danh mục chi
- [ ] Chuông thông báo nhận SSE realtime
- [ ] Dark mode, responsive, loading skeleton, empty state

**Xong khi:** dùng được app end-to-end trên trình duyệt, giao diện đủ đẹp để chụp màn hình đưa vào README.

---

### Phase 7 — Observability, CI/CD, Docs · 1 tuần
- [ ] Prometheus metrics: HTTP duration/status, consumer lag, số event xử lý/lỗi, độ trễ outbox
- [ ] Grafana dashboard (commit sẵn file JSON)
- [ ] OpenTelemetry tracing xuyên suốt HTTP → DB → Kafka header → worker, xem trên Jaeger
- [ ] GitHub Actions: lint → test (có service container) → build → docker build
- [ ] **README chuẩn CV**: mô tả 1 đoạn, sơ đồ kiến trúc, bảng tech stack, lý do chọn từng thứ, screenshot, hướng dẫn chạy trong 1 lệnh
- [ ] `docs/adr/` — vài ADR ngắn ghi lại quyết định thiết kế
- [ ] Load test bằng k6, ghi con số vào README

**Xong khi:** người lạ clone repo, chạy `make up`, có app hoạt động trong dưới 5 phút.

---

### Phase 8 — Deploy công khai · 4–5 ngày (tuỳ chọn)

> Phase này không bắt buộc. Xem mục 8 để cân nhắc có nên làm hay không.

**Ràng buộc quan trọng:** Vercel chỉ chạy được frontend Next.js. Go API và
các worker là tiến trình chạy liên tục, Kafka/PostgreSQL/Redis là dịch vụ
có trạng thái — Vercel không chạy được những thứ đó.

Cách deploy backend lên Vercel như ở dự án `heart_risk_estimator` (export
WSGI app của Django thành một serverless function) **không áp dụng được cho
dự án này**. Không phải vì ngôn ngữ, mà vì hình dạng hệ thống khác:

| | Ứng dụng hợp với serverless | Dự án này |
|---|---|---|
| Backend | Thuần request → response | API + 4 tiến trình chạy liên tục |
| Trạng thái | Đọc file model từ disk | Postgres + Redis + Kafka, cần ổ đĩa bền |
| Vòng đời | Sống vài giây rồi chết | Consumer phải sống mãi trong vòng lặp poll |

Serverless function bị kết thúc sau vài chục giây. Một Kafka consumer bị
kết thúc sau 60 giây thì không còn là consumer nữa. Tương tự, `outbox-relay`
là vòng lặp poll chứ không phải endpoint.

Vercel có runtime Go nên về lý thuyết bọc được Gin thành function, nhưng
mỗi lần gọi sẽ tạo connection pool mới làm cạn connection của Postgres, và
4 worker cùng Kafka broker vẫn không có chỗ chạy.

**Kiến trúc deploy:**

```
Vercel (web/)  ──https──>  api.<domain> (VPS)
                             └── docker compose: api, 3 worker,
                                 postgres, redis, kafka, caddy
```

**Việc làm:**
- [ ] `Dockerfile` multi-stage cho từng binary (build bằng golang, chạy trên
      distroless — image cuối khoảng 15MB)
- [ ] `docker-compose.prod.yml`: không expose port DB/Kafka ra internet,
      dùng Docker secret, thêm Caddy làm reverse proxy tự cấp SSL
- [ ] Chạy migration tự động lúc container khởi động
- [ ] Cấu hình `server.allowOrigins` trỏ tới domain Vercel thật
- [ ] Frontend đọc `NEXT_PUBLIC_API_URL` từ biến môi trường
- [ ] GitHub Actions thêm job deploy khi push lên main
- [ ] Backup định kỳ database

**Chi phí:** một VPS nhỏ khoảng 4–6 EUR/tháng là đủ chạy toàn bộ. Kafka
managed (Confluent, Redpanda Cloud, Aiven) đắt hơn nhiều và hầu như không
có free tier lâu dài — nên tự host Kafka trên chính VPS đó.

**Phương án 0 đồng:** bỏ qua phase này, thay bằng video demo 2 phút +
screenshot trong README + đảm bảo `docker compose up` chạy được ngay. Phần
lớn nhà tuyển dụng đọc README chứ không bấm link demo, và một link demo hay
sập còn phản tác dụng hơn là không có.

---

## 6. Tổng thời gian

| Phase | Thời lượng |
|---|---|
| 0. Nền móng | 3–4 ngày |
| 1. Data + Auth | 1 tuần |
| 2. Core domain | 1.5 tuần |
| 3. Kafka + Outbox | 1 tuần |
| 4. Analytics CQRS | 5–6 ngày |
| 5. Budget + Notification | 1 tuần |
| 6. Frontend | 2 tuần |
| 7. Observability + CI/CD | 1 tuần |
| 8. Deploy công khai (tuỳ chọn) | 4–5 ngày |
| **Tổng** | **~8 tuần part-time** (9 tuần nếu làm Phase 8) |

Nếu cần bản demo sớm để nộp CV: **Phase 0 → 3 → 6 (rút gọn)** cho ra một demo có Kafka + web trong ~3.5 tuần, rồi bổ sung dần.

## 7. Rủi ro & cách xử lý

| Rủi ro | Cách xử lý |
|---|---|
| Kafka trên Windows khó setup | Dùng KRaft mode 1 container, không Zookeeper; client `segmentio/kafka-go` thuần Go, không cần cgo |
| Sa lầy vào hạ tầng, không xong feature | Mỗi phase phải kết thúc bằng thứ chạy được; observability để cuối cùng |
| Frontend ngốn thời gian | Dùng shadcn/ui — copy component có sẵn, không tự viết design system |
| Test chậm do testcontainers | Tách `make test` (unit, nhanh) và `make itest` (integration), CI chạy cả hai |

## 8. Có nên deploy công khai không?

| Phương án | Chi phí | Ưu | Nhược |
|---|---|---|---|
| **A. VPS + Vercel** | ~4–6 EUR/tháng | Chạy được toàn bộ kiến trúc kể cả Kafka; tự deploy là điểm cộng | Phải tự lo SSL, firewall, backup |
| **B. PaaS từng service** (Railway/Render/Fly) | Đắt hơn A | Không phải quản trị server | 4 service chạy 24/7 + Kafka managed rất tốn; free tier hay cho service ngủ, demo load chậm |
| **C. Không deploy** | 0 đồng | Không có gì để hỏng | Không có link demo |

**Khuyến nghị: làm C trước, nâng lên A sau nếu muốn.**

Lý do: phần lớn nhà tuyển dụng đọc README và lướt code chứ không bấm link
demo. Một repo có sơ đồ kiến trúc rõ, video demo ngắn, screenshot đầy đủ và
chạy được bằng một lệnh sẽ được đánh giá cao hơn một link hosting rẻ hay
sập. Đừng để việc deploy chặn tiến độ code — nó là bước cuối cùng.

Lưu ý: giá và free tier của các dịch vụ managed Kafka thay đổi rất nhanh.
Hãy tự kiểm tra tại thời điểm deploy thay vì tin vào con số ghi trong tài
liệu này.
