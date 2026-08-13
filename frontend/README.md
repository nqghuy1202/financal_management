# Fina — Frontend

Giao diện quản lý tài chính cá nhân cho dự án `financal_management`.

**Stack:** React 19 + TypeScript + Vite 6 + Tailwind CSS v4 · Recharts · lucide-react · React Router 7.

## Yêu cầu

Node.js đã được cài sẵn (portable) tại `D:\tools\nodejs` — mọi thứ nằm trên ổ D.
Nếu `node` chưa có trong PATH của terminal mới, thêm `D:\tools\nodejs` vào PATH hoặc gọi trực tiếp bằng đường dẫn đầy đủ.

## Chạy dev

```powershell
$env:Path = "$env:Path;D:\tools\nodejs"
cd D:\financal_management\frontend
npm run dev
```

Mở http://localhost:5173

App gọi **API thật** nên cần backend chạy ở `http://localhost:8080` (Vite proxy `/api/*` sang đó) và MySQL.
Xem `../DEPLOY.md`: `docker compose up -d mysql_bp` rồi `make serve` (hoặc `docker compose up --build`).
Chưa có tài khoản demo — bấm **Đăng ký** để tạo tài khoản (hệ thống tự seed sẵn danh mục mặc định).

## Build production

```powershell
npm run build      # tsc -b && vite build  ->  dist/
npm run preview    # xem thử bản build
```

## Cấu trúc

```
src/
  components/    App shell (sidebar/topbar), Modal, StatCard, CategoryIcon, TransactionModal, ProtectedRoute
  context/       AuthContext (auth qua API + JWT), DataContext (CRUD giao dịch/danh mục/ngân sách qua API)
  lib/           api (client + JWT cho backend Gin), analytics, format
  pages/         Login, Register, Dashboard, Transactions, Budgets
  types.ts       Kiểu dữ liệu domain
```

## Dữ liệu & tích hợp backend

App đã **nối API thật** (không còn mock localStorage):

- `src/lib/api.ts` — client khớp envelope `{ code, message, data }`, lưu JWT ở localStorage (`fina.token`),
  gửi header `Authorization: Bearer`, ném `ApiError` (kèm `message`) khi request lỗi.
- `AuthContext` gọi `/api/auth/register|login|me`; `DataContext` tải + CRUD qua `/api/categories|transactions|budgets`.
- Dev server proxy `/api/*` sang `http://localhost:8080` (xem `vite.config.ts`); CORS backend cho phép origin `:5173`.
- Backend: package `internal/api` (MySQL + JWT). Xem `../DEPLOY.md` và `../CLAUDE.md`.
