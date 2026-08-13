# Triển khai — financal_management

App được đóng gói thành **một artifact duy nhất**: server Go (`cmd/web`) vừa phục vụ
API `/api/*` vừa phục vụ frontend React đã build (`frontend/dist`) kèm SPA fallback.

> Frontend hiện chạy trên dữ liệu giả (localStorage) nên app dùng được ngay mà chưa cần
> API/CSDL thật. MySQL trong compose đã sẵn cho bước nối API sau này.

## Biến môi trường

Sao chép `.env.example` → `.env` và chỉnh giá trị. Các biến chính:

| Biến | Mặc định | Ý nghĩa |
|------|----------|---------|
| `PORT` | `8080` | Cổng app phục vụ (API + FE) |
| `STATIC_DIR` | `./frontend/dist` | Thư mục chứa frontend đã build |
| `CORS_ORIGINS` | `http://localhost:5173` | Origin cho phép (phân tách bằng dấu phẩy) |
| `BLUEPRINT_DB_*` | — | Cấu hình MySQL (compose + API tương lai) |

## Cách 1 — Chạy trực tiếp (không Docker)

Yêu cầu Node (`D:\tools\nodejs`) và Go (`D:\tools\go\bin`) — xem `CLAUDE.md`.

```powershell
$env:Path = "$env:Path;D:\tools\nodejs;D:\tools\go\bin"
make serve          # = frontend-build + build-web + chạy web.exe
```

Hoặc từng bước:

```powershell
cd frontend; npm ci; npm run build; cd ..     # -> frontend/dist
go build -o web.exe ./cmd/web                  # build server
./web.exe                                      # phục vụ tại http://localhost:8080
```

Mở **http://localhost:8080** — cùng một cổng phục vụ cả UI lẫn API (`/api/health`).

## Cách 2 — Docker (khuyến nghị cho triển khai)

Dockerfile multi-stage: build FE bằng Node → build binary Go → image runtime Alpine nhỏ gọn.

```bash
# Chỉ mình app
docker build -t financal-management:latest .
docker run --rm -p 8080:8080 financal-management:latest

# Cả stack app + MySQL
cp .env.example .env        # rồi chỉnh mật khẩu
docker compose up --build -d
```

App: http://localhost:8080 · Health: http://localhost:8080/api/health

Dừng: `docker compose down` (thêm `-v` để xóa luôn dữ liệu MySQL).

## Kiểm tra nhanh

```bash
curl http://localhost:8080/api/health
# {"code":20000,"message":"OK","data":{"status":"up"}}
```

## Lưu ý kiến trúc

- `cmd/web/main.go` là entrypoint **production riêng**, tách khỏi các entrypoint thử nghiệm
  (`cmd/api` chặn mọi request bằng auth demo, `cmd/server` sai package). Deploy chỉ dùng `cmd/web`.
- Khi có API thật: mount route trong nhóm `/api` ở `cmd/web/main.go`, rồi thay lớp
  `frontend/src/lib/storage.ts` bằng `frontend/src/lib/api.ts`.
