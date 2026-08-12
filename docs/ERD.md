# Thiết kế dữ liệu

Tài liệu này ghi lại lược đồ dữ liệu và **lý do** đằng sau từng quyết định.
Migration thật nằm trong `migrations/`, được quản lý bằng goose.

## 1. Quyết định nền tảng

| Chủ đề | Lựa chọn | Lý do |
|---|---|---|
| Khoá chính | `uuid` (UUIDv7, sinh từ Go) | Tiền tố là timestamp nên vẫn tăng dần, index không phân mảnh như UUIDv4. Sinh được id **trước** khi ghi DB — cần thiết cho outbox pattern ở Phase 3. Không lộ số lượng bản ghi qua URL. |
| Kiểu tiền | `NUMERIC(19,4)` | Postgres tính toán chính xác tuyệt đối. Phía Go dùng `shopspring/decimal` bọc trong kiểu `model.Money`. |
| Thời gian | `timestamptz` | Luôn lưu UTC. Không bao giờ dùng `timestamp` (không có múi giờ) vì nó sẽ sai khi server và người dùng khác múi giờ. |
| Enum | `text` + `CHECK` | Kiểu `ENUM` của Postgres không xoá được giá trị đã thêm; mỗi lần đổi là một migration phiền phức. |
| Xoá dữ liệu tài chính | Soft delete (`deleted_at`) | Lịch sử giao dịch là dữ liệu không tái tạo được. Cũng cần cho việc replay event ở Phase 4. |
| Khoá ngoại | `ON DELETE RESTRICT` | Không bao giờ `CASCADE` trên dữ liệu tài chính — xoá một ví mà cuốn theo toàn bộ giao dịch là mất dữ liệu vĩnh viễn. |
| Đa tiền tệ | Mỗi ví một loại tiền, báo cáo không quy đổi | Có sẵn cột `currency` để mở rộng sau, nhưng chưa cần bảng tỷ giá. |

### Vì sao `NUMERIC` chứ không phải `BIGINT` đơn vị nhỏ nhất

Cả hai đều tránh được sai số dấu phẩy động. `NUMERIC` được chọn vì đọc dữ
liệu trực tiếp trong DB trực quan hơn (`12500.0000` thay vì `1250000`) và
không phải nhớ số chữ số thập phân của từng loại tiền.

Cái giá phải trả: Go không có kiểu decimal trong thư viện chuẩn. Rủi ro là
lập trình viên vô tình dùng `float64` cho tiền. Ta chặn bằng cách **không
bao giờ để lộ `decimal.Decimal` trần** ra ngoài — mọi số tiền đi qua kiểu
`model.Money` gắn liền `amount + currency`, và kiểu đó không có phương
thức nào nhận `float64`.

### Vì sao transfer là một dòng, không phải double-entry

Đây là ứng dụng quản lý chi tiêu cá nhân, không phải hệ thống kế toán.
Người dùng nghĩ "tôi chuyển 2 triệu từ ví tiền mặt sang tài khoản ngân
hàng" là **một** việc, nên danh sách giao dịch cũng nên hiện một dòng.
Double-entry đúng chuẩn kế toán hơn nhưng khiến mọi truy vấn danh sách và
báo cáo phải gộp lại theo cặp.

**Hệ quả phải nhớ:** mọi truy vấn báo cáo chi tiêu **bắt buộc** lọc
`type <> 'transfer'`. Nếu quên, chuyển tiền giữa hai ví của chính mình sẽ
bị đếm thành khoản chi. Đây là lỗi kinh điển của mô hình một dòng.

## 2. Lược đồ

```
                          ┌──────────────────────────┐
                          │         users            │
                          │ id  email  password_hash │
                          │ full_name  base_currency │
                          └────────────┬─────────────┘
                                       │ 1
         ┌──────────────┬──────────────┼──────────────┬──────────────┐
         │ N            │ N            │ N            │ N            │ N
   ┌─────▼──────┐ ┌─────▼──────┐ ┌─────▼──────┐ ┌─────▼─────┐ ┌──────▼────┐
   │  accounts  │ │ categories │ │transactions│ │  budgets  │ │   goals   │
   └─────▲──────┘ └─────▲──────┘ └─────┬──────┘ └───────────┘ └───────────┘
         │              │              │
         └──────────────┴──────────────┘
           account_id, counter_account_id, category_id
```

## 3. Bảng theo phase

Migration đi kèm code dùng đến nó, không tạo trước hàng loạt bảng rồi để
trống — đến lúc dùng gần như chắc chắn phải sửa lại thiết kế.

| Phase | Bảng |
|---|---|
| 1 | `users` |
| 2 | `accounts`, `categories`, `transactions` |
| 3 | `outbox_events`, `processed_events` |
| 4 | `rm_monthly_summary`, `rm_category_spending` |
| 5 | `budgets`, `goals`, `recurring_transactions`, `notifications` |

## 4. Chi tiết các bảng

### `users` (Phase 1)

| Cột | Kiểu | Ghi chú |
|---|---|---|
| `id` | `uuid` PK | UUIDv7 |
| `email` | `citext` UNIQUE | Không phân biệt hoa thường |
| `password_hash` | `text` | Argon2id, đã gồm salt và tham số |
| `full_name` | `text` | |
| `base_currency` | `char(3)` | ISO 4217, mặc định `VND` |
| `email_verified_at` | `timestamptz` NULL | |
| `created_at` / `updated_at` | `timestamptz` | |

Dùng `citext` thay vì `text` cho email: người dùng gõ `Huy@Gmail.com` hay
`huy@gmail.com` phải là cùng một tài khoản. Nếu dùng `text`, ràng buộc
UNIQUE sẽ cho phép đăng ký trùng chỉ khác hoa thường.

### `transactions` (Phase 2) — thiết kế trước để tham chiếu

| Cột | Ghi chú |
|---|---|
| `type` | `income` / `expense` / `transfer` |
| `amount` | `NUMERIC(19,4)`, **luôn dương**; dấu suy ra từ `type` |
| `occurred_at` | Ngày tiền thực sự chuyển, do người dùng nhập |
| `created_at` | Ngày ghi nhận vào hệ thống |
| `counter_account_id` | Chỉ có giá trị khi `type = 'transfer'` |

Tách `occurred_at` khỏi `created_at` là bắt buộc: hôm nay nhập bữa ăn của
hôm qua thì báo cáo tháng phải tính vào hôm qua.

### `categories` (Phase 2)

`user_id` cho phép `NULL`: `NULL` là danh mục hệ thống seed sẵn, có giá
trị là danh mục người dùng tự tạo. Vì `NULL` không so sánh bằng nhau
được trong ràng buộc UNIQUE, cần hai partial index riêng:

```sql
CREATE UNIQUE INDEX ON categories (user_id, lower(name), type) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX ON categories (lower(name), type)          WHERE user_id IS NULL;
```

### `outbox_events` (Phase 3)

Index quyết định hiệu năng của relay là partial index chỉ chứa event chưa
gửi — nó luôn nhỏ dù bảng có hàng triệu dòng:

```sql
CREATE INDEX ON outbox_events (created_at) WHERE published_at IS NULL;
```

## 5. Refresh token không nằm trong PostgreSQL

Refresh token lưu trong Redis với TTL bằng đúng thời hạn của token, nên
token hết hạn tự biến mất mà không cần job dọn dẹp.

Kèm phát hiện tái sử dụng: mỗi refresh token mang một `jti`, dùng xong là
xoá ngay. Nếu ai đó trình lại một `jti` đã bị xoá, nghĩa là token đã bị
đánh cắp và dùng lại — khi đó revoke toàn bộ phiên đăng nhập của người
dùng đó.

## 6. Số dư ví

`accounts.balance` được lưu sẵn và cập nhật trong **cùng transaction** với
việc ghi giao dịch, dùng `SELECT ... FOR UPDATE` để tránh cập nhật chồng
chéo khi có hai request đồng thời.

Số dư lưu sẵn có thể lệch nếu code sai, nên kèm một truy vấn đối soát tính
lại số dư từ bảng `transactions` để so sánh. Đọc nhanh mà vẫn kiểm chứng
được tính đúng đắn.
