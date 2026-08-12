-- Danh mục có hai loại sống chung trong một bảng:
--   user_id IS NULL  → danh mục hệ thống, mọi người dùng đều thấy
--   user_id có giá trị → danh mục riêng của người đó
--
-- Quy tắc: ai cũng ĐỌC được danh mục hệ thống, nhưng chỉ SỬA/XOÁ được
-- danh mục của chính mình. Quy tắc đó được cài đặt bằng điều kiện WHERE
-- chứ không phải bằng if trong Go: câu UPDATE và DELETE dùng
-- `user_id = @user_id`, mà danh mục hệ thống có user_id NULL nên không
-- bao giờ khớp — tự động được bảo vệ.

-- name: ListCategories :many
SELECT * FROM categories
WHERE (user_id = @user_id OR user_id IS NULL)
  AND deleted_at IS NULL
  -- Bỏ trống tham số type thì lấy cả thu lẫn chi.
  AND (sqlc.narg('type')::text IS NULL OR type = sqlc.narg('type'))
-- Danh mục hệ thống hiện trước, sau đó xếp theo tên.
ORDER BY user_id NULLS FIRST, name;

-- name: GetCategory :one
-- Đọc được cả danh mục hệ thống lẫn danh mục của chính mình.
SELECT * FROM categories
WHERE id = @id
  AND (user_id = @user_id OR user_id IS NULL)
  AND deleted_at IS NULL;

-- name: CreateCategory :one
INSERT INTO categories (id, user_id, name, type, icon, color)
VALUES (@id, @user_id, @name, @type, @icon, @color)
RETURNING *;

-- name: UpdateCategory :one
-- Chỉ sửa được danh mục của chính mình: danh mục hệ thống có user_id
-- NULL nên điều kiện dưới đây không bao giờ khớp.
UPDATE categories
SET name  = @name,
    icon  = @icon,
    color = @color
WHERE id = @id
  AND user_id = @user_id
  AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteCategory :one
UPDATE categories
SET deleted_at = now()
WHERE id = @id
  AND user_id = @user_id
  AND deleted_at IS NULL
RETURNING *;

-- name: CountTransactionsByCategory :one
-- Dùng trước khi xoá: danh mục còn giao dịch thì không cho xoá, vì xoá đi
-- sẽ làm các báo cáo cũ mất một phần dữ liệu.
SELECT count(*) FROM transactions
WHERE category_id = @category_id
  AND deleted_at IS NULL;
