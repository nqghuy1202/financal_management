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
| Khoá ngoại | `ON DELETE RESTRICT` | Không bao giờ `CASCADE` trên dữ liệu tài chính — xoá một nguồn tiền mà cuốn theo toàn bộ giao dịch là mất dữ liệu vĩnh viễn. |
| Đa tiền tệ | Mỗi giao dịch mang loại tiền riêng, báo cáo không quy đổi | Có sẵn cột `currency` để mở rộng sau, nhưng chưa cần bảng tỷ giá. |

### Vì sao `NUMERIC` chứ không phải `BIGINT` đơn vị nhỏ nhất

Cả hai đều tránh được sai số dấu phẩy động. `NUMERIC` được chọn vì đọc dữ
liệu trực tiếp trong DB trực quan hơn (`12500.0000` thay vì `1250000`) và
không phải nhớ số chữ số thập phân của từng loại tiền.

Cái giá phải trả: Go không có kiểu decimal trong thư viện chuẩn. Rủi ro là
lập trình viên vô tình dùng `float64` cho tiền. Ta chặn bằng cách **không
bao giờ để lộ `decimal.Decimal` trần** ra ngoài — mọi số tiền đi qua kiểu
`model.Money` gắn liền `amount + currency`, và kiểu đó không có phương
thức nào nhận `float64`.

### Vì sao không có chức năng chuyển khoản

Trọng tâm của ứng dụng là **tổng hợp bức tranh tài chính**: người dùng
nhập các khoản thu và chi, hệ thống tính toán và trả lại cái nhìn tổng
quan theo tháng hoặc năm.

Chuyển tiền giữa hai nguồn của chính mình không làm thay đổi bức tranh đó
— tổng thu không đổi, tổng chi không đổi. Thêm chức năng này chỉ làm mọi
truy vấn báo cáo phải nhớ loại trừ nó, và quên một chỗ là số liệu sai.

Hệ quả: bảng `accounts` chỉ còn là **nhãn nguồn tiền** (tiền mặt, thẻ,
ví điện tử), không có cột số dư. Số dư ròng được tính từ toàn bộ giao
dịch khi làm báo cáo, nên không bao giờ lệch.

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
   ┌─────▼──────┐ ┌─────▼──────┐ ┌─────▼──────┐ ┌─────▼─────┐
   │  accounts  │ │ categories │ │transactions│ │  budgets  │
   │ (nguồn tiền)│ │            │ │            │ │ (Phase 5) │
   └─────▲──────┘ └─────▲──────┘ └─────┬──────┘ └───────────┘
         │              │              │
         └──────────────┴──────────────┘
            account_id, category_id (đều cho phép NULL)
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
| 5 | `budgets`, `notifications` |

## 4. Chi tiết các bảng

### `users` (Phase 1)

| Cột | Kiểu | Ghi chú |
|---|---|---|
| `id` | `uuid` PK | UUIDv7 |
| `email` | `citext` UNIQUE | Không phân biệt hoa thường |
| `password_hash` | `text` | bcrypt cost 12, chuỗi đã gồm sẵn salt |
| `full_name` | `text` | |
| `base_currency` | `char(3)` | ISO 4217, mặc định `VND` |
| `email_verified_at` | `timestamptz` NULL | |
| `created_at` / `updated_at` | `timestamptz` | |

Dùng `citext` thay vì `text` cho email: người dùng gõ `Huy@Gmail.com` hay
`huy@gmail.com` phải là cùng một tài khoản. Nếu dùng `text`, ràng buộc
UNIQUE sẽ cho phép đăng ký trùng chỉ khác hoa thường.

### `transactions` (Phase 2)

| Cột | Ghi chú |
|---|---|
| `type` | Chỉ `income` hoặc `expense` |
| `amount` | `NUMERIC(19,4)`, **luôn dương**; dấu suy ra từ `type` |
| `occurred_at` | Ngày tiền thực sự thu/chi, do người dùng nhập |
| `created_at` | Ngày ghi nhận vào hệ thống |
| `account_id` | Nguồn tiền, cho phép NULL |
| `category_id` | Danh mục, cho phép NULL |

Tách `occurred_at` khỏi `created_at` là bắt buộc: hôm nay nhập bữa ăn của
hôm qua thì báo cáo tháng phải tính vào hôm qua. **Mọi báo cáo tổng hợp
theo `occurred_at`**, không bao giờ theo `created_at`.

`account_id` và `category_id` đều cho phép NULL để người dùng ghi nhanh
"hôm nay ăn trưa 50k" mà không phải chọn thêm gì. Báo cáo gom các khoản
không có danh mục vào nhóm "Chưa phân loại".

### `categories` (Phase 2)

Danh mục **phẳng, không phân cấp**. Bản thiết kế đầu có cột `parent_id`
cho danh mục con, nhưng đã bỏ: truy vấn cây cần `WITH RECURSIVE`, làm mọi
câu lệnh lọc và báo cáo phức tạp hơn hẳn trong khi giá trị thêm cho một
app chi tiêu cá nhân là nhỏ. Nếu sau này cần, thêm cột là một migration
đơn giản.

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

Refresh token là một chuỗi ngẫu nhiên 32 byte, lưu trong Redis với TTL
bằng đúng thời hạn của token, nên token hết hạn tự biến mất mà không cần
job dọn dẹp.

Mỗi token chỉ dùng được một lần: `ConsumeRefresh` gọi `GETDEL` của Redis,
đọc và xoá trong cùng một lệnh. Nhờ vậy hai request đồng thời không thể
cùng đổi một token, và token đã dùng thì không dùng lại được.

Bản thiết kế đầu có thêm cơ chế phát hiện tái sử dụng (trình lại token đã
xoá thì thu hồi toàn bộ phiên của người dùng đó), nhưng đã bỏ cho đơn
giản. Có thể thêm sau mà không phải đổi cấu trúc dữ liệu.

## 6. Danh mục hệ thống được seed bằng migration

16 danh mục mặc định (Ăn uống, Đi lại, Lương...) có `user_id = NULL` và
được chèn một lần bằng migration, không sinh lại cho từng người dùng khi
đăng ký.

Lý do: không nhân bản hàng chục dòng giống nhau cho mỗi tài khoản mới,
đăng ký chỉ còn một thao tác ghi nên không cần transaction nhiều bảng, và
sửa tên hay biểu tượng của một danh mục chỉ là sửa một dòng.

Người dùng vẫn tạo được danh mục riêng; khi đó `user_id` có giá trị.
