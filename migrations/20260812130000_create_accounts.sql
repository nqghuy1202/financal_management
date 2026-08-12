-- +goose Up
-- +goose StatementBegin

-- Bảng này ghi nhận NGUỒN TIỀN của mỗi giao dịch: tiền mặt, thẻ ngân
-- hàng, ví điện tử...
--
-- Cố tình KHÔNG có cột số dư. Ứng dụng chỉ ghi nhận thu và chi rồi tổng
-- hợp lại, không có chức năng chuyển tiền giữa các nguồn. Nếu lưu số dư
-- riêng cho từng nguồn thì tiền sẽ bị "kẹt" trong từng nguồn và con số
-- đó nhanh chóng lệch khỏi thực tế. Số dư ròng được tính từ toàn bộ
-- giao dịch khi làm báo cáo.
CREATE TABLE accounts (
    id         UUID        PRIMARY KEY,

    -- RESTRICT chứ không CASCADE: xoá người dùng mà cuốn theo toàn bộ
    -- lịch sử chi tiêu là mất dữ liệu không phục hồi được.
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,

    name       TEXT        NOT NULL,

    -- cash: tiền mặt, bank: tài khoản ngân hàng,
    -- ewallet: ví điện tử (Momo, ZaloPay), credit_card: thẻ tín dụng.
    type       TEXT        NOT NULL,

    icon       TEXT        NOT NULL DEFAULT '',

    -- Soft delete: giữ lại bản ghi để các giao dịch cũ vẫn tham chiếu được.
    deleted_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT accounts_name_not_blank CHECK (length(trim(name)) > 0),
    CONSTRAINT accounts_type_valid     CHECK (type IN ('cash', 'bank', 'ewallet', 'credit_card'))
);

-- Trong một tài khoản không được có hai nguồn tiền trùng tên.
-- Partial index để nguồn đã xoá mềm không chiếm chỗ tên đó nữa.
CREATE UNIQUE INDEX accounts_user_name_key
    ON accounts (user_id, lower(name))
    WHERE deleted_at IS NULL;

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
