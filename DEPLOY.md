# Triển khai — financal_management

App được đóng gói thành **một artifact duy nhất**: server Go (`cmd/web`) vừa phục vụ
API `/api/*` vừa phục vụ frontend React đã build (`frontend/dist`) kèm SPA fallback.

Frontend gọi API thật (`frontend/src/lib/api.ts`) — auth, transactions, budgets, fixed costs,
safe-to-spend đều đọc/ghi MySQL qua `internal/api`, không còn dùng dữ liệu giả localStorage.

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

## Cách 3 — VPS production (dùng chung VPS debt-crusher)

Domain: **finance.hlcompany.id.vn**. Dùng chung VPS đã chạy debt-crusher — Docker, Nginx, Certbot,
UFW đã cài sẵn từ lần deploy trước, **không cần cài lại**. Kiến trúc:

```
Browser → Nginx (host, :80/:443, server_name finance.hlcompany.id.vn)
        → 127.0.0.1:8081 (container financal-app, cổng nội bộ 8080)
        → MySQL (container financal-mysql, chỉ nội bộ)
```

Cổng host **8081/3307** (khác 8080/3306 debt-crusher đang chiếm) để tránh đụng port trên cùng VPS —
cổng bên trong container vẫn 8080/3306 như bình thường.

### 1. Trỏ domain

Thêm bản ghi DNS tại nơi quản lý `hlcompany.id.vn`:

| Type | Name | Value |
|------|------|-------|
| A    | `finance` | `<IP_VPS>` (IP đã dùng cho debt-crusher) |

Kiểm tra đã lan truyền trước khi xin SSL: `ping finance.hlcompany.id.vn`.

### 2. Đưa code lên VPS

```bash
git clone <URL_repo> /opt/financal_management
cd /opt/financal_management
```

(hoặc `git pull` nếu đã clone từ trước.)

### 3. Cấu hình secrets

```bash
cp .env.prod.example .env.prod
nano .env.prod   # điền JWT_SECRET, BLUEPRINT_DB_PASSWORD, BLUEPRINT_DB_ROOT_PASSWORD — sinh bằng: openssl rand -hex 32
```

Dùng giá trị **khác** với `.env.prod` của debt-crusher — không tái sử dụng secrets giữa 2 app dù
chạy chung VPS.

### 4. Build & chạy

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
docker compose -f docker-compose.prod.yml ps     # cả 2 service phải "running"/"healthy"
docker compose -f docker-compose.prod.yml logs -f app   # Ctrl+C để thoát, kiểm tra app boot OK
```

### 5. Trỏ Nginx vào container

```bash
cp /opt/financal_management/nginx/finance.hlcompany.id.vn.conf /etc/nginx/sites-available/financal
ln -s /etc/nginx/sites-available/financal /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx
```

Lúc này `http://finance.hlcompany.id.vn` đã lên được (chưa có HTTPS).

### 6. Bật HTTPS

Certbot đã cài (từ lần deploy debt-crusher) — chỉ cần xin thêm chứng chỉ cho subdomain mới:

```bash
certbot --nginx -d finance.hlcompany.id.vn
```

Certbot tự sửa `/etc/nginx/sites-available/financal`: thêm block 443 + redirect 80→443, tự gia hạn.

### 7. Kiểm tra

```bash
curl https://finance.hlcompany.id.vn/api/health
```

### Cập nhật code sau này

```bash
cd /opt/financal_management && git pull
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
```

`--build` rebuild lại image nếu code đổi; MySQL data ở volume `./data/db_data` không mất.

### Troubleshooting

- **502 Bad Gateway**: container `financal-app` chưa chạy/chưa healthy — check
  `docker compose -f docker-compose.prod.yml ps` và `... logs app`.
- **`certbot --nginx` báo domain không resolve**: DNS A record chưa trỏ đúng/chưa kịp lan truyền.
- **Đổi `JWT_SECRET`/`BLUEPRINT_DB_PASSWORD` sau khi đã chạy**: cần
  `docker compose -f docker-compose.prod.yml up -d --force-recreate app mysql`; đổi mật khẩu DB sau
  khi MySQL đã init lần đầu thì phải tự `ALTER USER` trong MySQL, vì biến `MYSQL_PASSWORD` chỉ áp
  dụng lúc tạo user lần đầu.
- Debug MySQL qua SSH tunnel khi cần: `ssh -L 3307:127.0.0.1:3307 root@<IP_VPS>` (container không
  map port ra ngoài internet nên tunnel này an toàn).

## Lưu ý kiến trúc

- `cmd/web/main.go` là entrypoint **production riêng**, tách khỏi các entrypoint thử nghiệm
  (`cmd/api` chặn mọi request bằng auth demo, `cmd/server` sai package). Deploy chỉ dùng `cmd/web`.
- Khi có API thật: mount route trong nhóm `/api` ở `cmd/web/main.go`, rồi thay lớp
  `frontend/src/lib/storage.ts` bằng `frontend/src/lib/api.ts`.
