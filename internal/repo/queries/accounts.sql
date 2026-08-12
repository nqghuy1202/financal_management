-- Mọi truy vấn ở đây đều có điều kiện user_id.
--
-- Đây là điểm bảo mật quan trọng nhất của tầng dữ liệu: nếu chỉ lọc theo
-- id, người dùng A chỉ cần đoán được id ví của người dùng B là đọc hoặc
-- sửa được ví đó. Ràng buộc user_id ngay trong câu SQL khiến không handler
-- nào có thể quên kiểm tra quyền sở hữu.

-- name: CreateAccount :one
INSERT INTO accounts (id, user_id, name, type, currency, balance, icon)
VALUES (@id, @user_id, @name, @type, @currency, @balance, @icon)
RETURNING *;

-- name: GetAccount :one
SELECT * FROM accounts
WHERE id = @id
  AND user_id = @user_id
  AND deleted_at IS NULL;

-- name: ListAccounts :many
SELECT * FROM accounts
WHERE user_id = @user_id
  AND deleted_at IS NULL
ORDER BY created_at;

-- name: UpdateAccount :one
UPDATE accounts
SET name = @name,
    type = @type,
    icon = @icon
WHERE id = @id
  AND user_id = @user_id
  AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteAccount :one
-- Xoá mềm để các giao dịch cũ vẫn tham chiếu được tới ví này.
UPDATE accounts
SET deleted_at = now()
WHERE id = @id
  AND user_id = @user_id
  AND deleted_at IS NULL
RETURNING *;

-- name: CountTransactionsByAccount :one
-- Dùng trước khi xoá ví: ví còn giao dịch thì cảnh báo người dùng.
SELECT count(*) FROM transactions
WHERE (account_id = @account_id OR counter_account_id = @account_id)
  AND deleted_at IS NULL;

-- name: GetAccountForUpdate :one
-- Khoá dòng ví lại trước khi đổi số dư.
--
-- FOR UPDATE khiến transaction khác muốn khoá cùng dòng phải xếp hàng
-- chờ. Nếu không có nó, hai giao dịch đồng thời trên cùng một ví có thể
-- cùng đọc số dư cũ rồi cùng ghi đè, làm mất một trong hai khoản tiền.
SELECT * FROM accounts
WHERE id = @id
  AND user_id = @user_id
  AND deleted_at IS NULL
FOR UPDATE;

-- name: UpdateAccountBalance :exec
UPDATE accounts
SET balance = @balance
WHERE id = @id;
