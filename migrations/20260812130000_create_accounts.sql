-- +goose Up
-- +goose StatementBegin

CREATE TABLE accounts (
    id         UUID          PRIMARY KEY,

    -- RESTRICT chứ không CASCADE: xoá người dùng mà cuốn theo toàn bộ ví
    -- và lịch sử giao dịch là mất dữ liệu không phục hồi được.
    user_id    UUID          NOT NULL REFERENCES users (id) ON DELETE RESTRICT,

    name       TEXT          NOT NULL,

    -- cash: tiền mặt, bank: tài khoản ngân hàng,
    -- ewallet: ví điện tử (Momo, ZaloPay), credit_card: thẻ tín dụng.
    type       TEXT          NOT NULL,

    currency   CHAR(3)       NOT NULL DEFAULT 'VND',

    -- Số dư được lưu sẵn và cập nhật trong cùng transaction với việc ghi
    -- giao dịch. Nếu tính lại từ bảng transactions mỗi lần đọc thì màn
    -- hình danh sách ví sẽ chậm dần theo số giao dịch.
    balance    NUMERIC(19,4) NOT NULL DEFAULT 0,

    icon       TEXT          NOT NULL DEFAULT '',

    -- Soft delete: giữ lại bản ghi để các giao dịch cũ vẫn tham chiếu được.
    deleted_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT now(),

    CONSTRAINT accounts_name_not_blank CHECK (length(trim(name)) > 0),
    CONSTRAINT accounts_type_valid     CHECK (type IN ('cash', 'bank', 'ewallet', 'credit_card')),
    CONSTRAINT accounts_currency_format CHECK (currency ~ '^[A-Z]{3}$')
);

-- Trong một tài khoản không được có hai ví trùng tên.
-- Partial index để ví đã xoá mềm không chiếm chỗ tên đó nữa.
CREATE UNIQUE INDEX accounts_user_name_key
    ON accounts (user_id, lower(name))
    WHERE deleted_at IS NULL;

-- Truy vấn phổ biến nhất: liệt kê ví của một người dùng.
CREATE INDEX accounts_user_id_idx
    ON accounts (user_id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER accounts_set_updated_at
    BEFORE UPDATE ON accounts
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS accounts;
-- +goose StatementEnd
