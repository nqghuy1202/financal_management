# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Toolchain (không có sẵn trên PATH)

Node và Go được cài **portable trên ổ D** (không nằm trên C, không có trong PATH của shell mới):

- Node: `D:\tools\nodejs` (v22.x). Trong shell mới: `$env:Path = "$env:Path;D:\tools\nodejs"` hoặc gọi thẳng `& 'D:\tools\nodejs\npm.cmd'`.
- Go: yêu cầu `go 1.26.x` (xem `go.mod`). Nếu chưa cài, bản portable đặt ở `D:\tools\go\bin`.
- Shell chính là PowerShell (Windows). Makefile dùng nhiều cú pháp Windows (`main.exe`, `powershell` trong target `watch`).

## Commands

### Backend (chạy từ gốc repo)

- Build: `make build` → `go build -o main.exe cmd/api/main.go`
- Run: `make run` → `go run cmd/api/main.go` (phục vụ **:8080** qua `routers.NewRouter()`)
- Test tất cả: `make test` → `go test ./... -v`
- Test một package: `go test ./internal/database -v`
- Test một hàm: `go test ./... -run <TenTest> -v`
- Integration test DB (testcontainers, cần Docker): `make itest` → `go test ./internal/database -v`
- Live reload (air): `make watch`
- MySQL bằng Docker: `make docker-run` / `make docker-down` (cần `.env` với các biến `BLUEPRINT_DB_*`)

### Frontend (`frontend/`)

- Dev: `npm run dev` → **:5173**, proxy mọi request `/api/*` sang `http://localhost:8080` (xem `frontend/vite.config.ts`)
- Build production: `npm run build` → `tsc -b && vite build` xuất ra `frontend/dist/`
- Type-check/lint: `npm run lint` → `tsc -b --noEmit`
- Tài khoản demo trên màn hình đăng nhập: `demo@fina.vn` / `123456`

## Kiến trúc

### Backend (Go + Gin) — có NHIỀU entrypoint mâu thuẫn (quan trọng)

Đây là scaffold học tập; hiện diện ba luồng khởi động khác nhau, cần biết cái nào thực sự chạy:

1. **`cmd/api/main.go` — luồng ĐANG chạy** (Makefile build/run trỏ vào đây). Gọi `routers.NewRouter()` → Gin ở **:8080** và áp `middlewares.AuthenMiddleware()` lên **mọi route**. Middleware này (demo) **chặn** request trừ khi có header `Authorization: invalid-token`, và router này **không có CORS**. Vì vậy nó chưa phục vụ được frontend như hiện trạng.
2. **`internal/server`** (`server.go` + `routes.go`): một `Server` sạch hơn có CORS cho `http://localhost:5173` và `/health`, nhưng **không được main nào dùng** (phần này bị comment trong `cmd/api/main.go`).
3. **`internal/initialize.Run()`**: bootstrap đầy đủ (`LoadConfig → InitLogger → InitMySQL → InitRedis → InitRouter`) chạy ở **:8002**. Nhưng `cmd/server/main.go` khai báo `package server` (không phải `package main`) nên **không chạy được như entrypoint**.

**API THẬT đang dùng — `internal/api`** (tách khỏi scaffold `controller/service/repo` còn dở): package tự chứa gồm `db.go` (kết nối MySQL từ `BLUEPRINT_DB_*` + auto-migrate bảng `users/categories/transactions/budgets` khi boot), `auth.go` (bcrypt + JWT HS256 từ `JWT_SECRET`, middleware `Authorization: Bearer`), và handler CRUD `categories.go/transactions.go/budgets.go`. `cmd/web/main.go` gọi `api.Connect()`→`api.Migrate()`→`api.NewHandler(db, secret).Register(r.Group("/api"))`. Route công khai: `/api/auth/register`, `/api/auth/login`; còn lại yêu cầu JWT. Đăng ký sẽ seed sẵn 10 danh mục mặc định cho user. **Chạy full stack cần MySQL + `.env`** (xem `.env.example`, `DEPLOY.md`): `docker compose up -d mysql_bp` rồi `make serve` (hoặc `docker compose up --build`).

**Mẫu phân tầng của scaffold cũ (KHÔNG dùng cho API thật):** `controller → service → repo → model → database` (xem `internal/controller`, `internal/services`, `internal/repo`).

**Response envelope:** mọi API trả `response.ResponseData{ Code, Message, Data }` (JSON: `{code, message, data}`). `Message` tra từ map `msg[code]` trong `internal/pkg/response/httpStatusCode.go`. Dùng `response.SuccessResponse` / `response.ErrResponse`.

**Cấu hình:** Viper đọc `config/{local|production}.yaml` (`internal/initialize/loadconfig.go`), struct trong `internal/pkg/setting`. `godotenv/autoload` nạp `.env` (biến `PORT`, `BLUEPRINT_DB_*`). Singleton dùng chung (config, logger, mysql, redis) đặt ở `global/global.go`.

### Frontend (React 19 + TypeScript + Vite 6 + Tailwind v4)

Thư mục `frontend/` (chi tiết ở `frontend/README.md`). Điểm mấu chốt về kiến trúc:

- **Đã nối API thật:** frontend gọi backend qua **`src/lib/api.ts`** (JWT lưu ở localStorage key `fina.token`, gửi header `Authorization: Bearer`; parse envelope `{code, message, data}` và ném `ApiError` khi lỗi). Lớp mock `storage.ts`/`seed.ts` **đã bị xóa**.
- **State qua Context:** `context/AuthContext.tsx` (register/login/logout/khôi phục phiên qua `/api/auth/*`, có cờ `loading`) và `context/DataContext.tsx` (tải + CRUD giao dịch/danh mục/ngân sách qua API theo user; tính toán phái sinh trong `src/lib/analytics.ts`). `ProtectedRoute` chờ `loading` trước khi điều hướng.
- **i18n Anh/Việt:** `context/I18nContext.tsx` + `src/lib/i18n.ts` (từ điển `vi`/`en`, hàm `t(key, vars)`, `localizedMonth`, `localizedCategory`). Nút `LanguageToggle` (VI/EN, lưu `fina.lang`). Tên danh mục mặc định được localize tập trung trong `DataContext` (map theo `lang`; danh mục người dùng tạo giữ nguyên). **Lưu ý:** nhiều file dùng biến `t` cho transaction → alias hook i18n thành `tr` để tránh trùng.
- **Toast:** `context/ToastContext.tsx` (`useToast().success/error`); mọi thao tác CRUD bắt lỗi và báo toast.
- **Trang:** `Login`, `Register`, `Dashboard`, `Transactions` (bảng có nhóm/tìm/ẩn-hiện cột qua một nút "Tùy chọn"; số tiền luôn dương, tổng = Thu − Chi), `Budgets`.
- **Quy ước UI:** thương hiệu **HL Company**, logo monogram đơn sắc `src/components/Logo.tsx` dùng `currentColor`. Chỉ dùng **icon đơn sắc** (lucide) recolor theo hệ thống — **không** dùng emoji làm icon, **không** tô màu lên icon danh mục (màu category chỉ dành cho biểu đồ/chấm dữ liệu). Token màu định nghĩa trong `src/index.css` (`@theme`): `brand-*` (xanh) và `ink-*` (xám). Dùng các class tiện ích `.btn-primary/.btn-outline/.btn-icon/.input/.card/.chip/.tnum`.
