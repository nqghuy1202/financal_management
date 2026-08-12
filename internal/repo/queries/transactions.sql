-- name: CreateTransaction :one
INSERT INTO transactions (
    id, user_id, account_id, category_id, type, amount, currency, note, occurred_at
) VALUES (
    @id, @user_id, @account_id, @category_id, @type, @amount, @currency, @note, @occurred_at
)
RETURNING *;

-- name: GetTransaction :one
SELECT * FROM transactions
WHERE id = @id
  AND user_id = @user_id
  AND deleted_at IS NULL;

-- name: UpdateTransaction :one
UPDATE transactions
SET account_id  = @account_id,
    category_id = @category_id,
    type        = @type,
    amount      = @amount,
    note        = @note,
    occurred_at = @occurred_at
WHERE id = @id
  AND user_id = @user_id
  AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteTransaction :one
UPDATE transactions
SET deleted_at = now()
WHERE id = @id
  AND user_id = @user_id
  AND deleted_at IS NULL
RETURNING *;

-- name: ListTransactions :many
--
-- Phân trang bằng CON TRỎ chứ không dùng OFFSET.
--
-- OFFSET có hai vấn đề: database vẫn phải đọc và bỏ đi toàn bộ số dòng
-- được bỏ qua nên trang càng sau càng chậm; và nếu có giao dịch mới được
-- thêm giữa hai lần gọi thì các trang sau bị lệch, người dùng thấy lặp
-- hoặc mất bản ghi.
--
-- Con trỏ ở đây là cặp (occurred_at, id). Cần cả hai vì nhiều giao dịch
-- có thể trùng ngày; id là chốt chặn để thứ tự luôn xác định. Phép so
-- sánh bộ (a, b) < (c, d) của Postgres so sánh theo thứ tự từ trái sang,
-- đúng như thứ tự sắp xếp.
--
-- Các tham số lọc dùng sqlc.narg: để NULL nghĩa là không lọc theo tiêu
-- chí đó.
SELECT * FROM transactions
WHERE user_id = @user_id
  AND deleted_at IS NULL
  AND (sqlc.narg('type')::text        IS NULL OR type        = sqlc.narg('type'))
  AND (sqlc.narg('category_id')::uuid IS NULL OR category_id = sqlc.narg('category_id'))
  AND (sqlc.narg('account_id')::uuid  IS NULL OR account_id  = sqlc.narg('account_id'))
  AND (sqlc.narg('from')::timestamptz IS NULL OR occurred_at >= sqlc.narg('from'))
  AND (sqlc.narg('to')::timestamptz   IS NULL OR occurred_at <  sqlc.narg('to'))
  AND (
        sqlc.narg('cursor_occurred_at')::timestamptz IS NULL
        OR (occurred_at, id) < (sqlc.narg('cursor_occurred_at'), sqlc.narg('cursor_id')::uuid)
      )
ORDER BY occurred_at DESC, id DESC
LIMIT @page_size;

-- name: CountTransactions :one
-- Tổng số giao dịch khớp bộ lọc, để frontend hiện "có N kết quả".
SELECT count(*) FROM transactions
WHERE user_id = @user_id
  AND deleted_at IS NULL
  AND (sqlc.narg('type')::text        IS NULL OR type        = sqlc.narg('type'))
  AND (sqlc.narg('category_id')::uuid IS NULL OR category_id = sqlc.narg('category_id'))
  AND (sqlc.narg('account_id')::uuid  IS NULL OR account_id  = sqlc.narg('account_id'))
  AND (sqlc.narg('from')::timestamptz IS NULL OR occurred_at >= sqlc.narg('from'))
  AND (sqlc.narg('to')::timestamptz   IS NULL OR occurred_at <  sqlc.narg('to'));
