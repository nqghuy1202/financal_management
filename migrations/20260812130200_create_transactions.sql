-- +goose Up
-- +goose StatementBegin

CREATE TABLE transactions (
    id                 UUID          PRIMARY KEY,
    user_id            UUID          NOT NULL REFERENCES users (id)      ON DELETE RESTRICT,

    -- Ví bị trừ tiền (chi, chuyển đi) hoặc được cộng tiền (thu).
    account_id         UUID          NOT NULL REFERENCES accounts (id)   ON DELETE RESTRICT,

    -- Ví nhận tiền. Chỉ dùng khi type = 'transfer'.
    counter_account_id UUID          REFERENCES accounts (id)            ON DELETE RESTRICT,

    -- Chuyển khoản giữa hai ví của chính mình không phải là thu hay chi,
    -- nên không gắn danh mục.
    category_id        UUID          REFERENCES categories (id)          ON DELETE RESTRICT,

    type               TEXT          NOT NULL,

    -- Luôn là số dương. Tiền vào hay ra được suy ra từ cột type, không
    -- dùng dấu âm — nếu dùng dấu, mọi câu tổng hợp đều phải nhớ xử lý
    -- dấu và rất dễ sai.
    amount             NUMERIC(19,4) NOT NULL,

    -- Lưu lại loại tiền tại thời điểm giao dịch. Không đọc từ bảng
    -- accounts vì ví có thể đổi loại tiền về sau, khi đó lịch sử cũ vẫn
    -- phải giữ đúng loại tiền lúc phát sinh.
    currency           CHAR(3)       NOT NULL,

    note               TEXT          NOT NULL DEFAULT '',

    -- Thời điểm tiền thực sự chuyển, do người dùng nhập.
    -- Khác created_at là thời điểm bản ghi được tạo trong hệ thống: hôm
    -- nay nhập bữa ăn của hôm qua thì báo cáo phải tính vào hôm qua.
    occurred_at        TIMESTAMPTZ   NOT NULL,

    deleted_at         TIMESTAMPTZ,

    created_at         TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ   NOT NULL DEFAULT now(),

    CONSTRAINT transactions_type_valid     CHECK (type IN ('income', 'expense', 'transfer')),
    CONSTRAINT transactions_amount_positive CHECK (amount > 0),
    CONSTRAINT transactions_currency_format CHECK (currency ~ '^[A-Z]{3}$'),

    -- Ràng buộc quan trọng nhất của bảng này: mỗi loại giao dịch phải có
    -- đúng bộ cột của nó. Đặt ở database để dù có đường ghi dữ liệu nào
    -- bỏ qua tầng ứng dụng thì cũng không tạo ra được bản ghi vô nghĩa
    -- kiểu "chuyển khoản mà không có ví đích".
    CONSTRAINT transactions_shape_valid CHECK (
        (type = 'transfer'
            AND counter_account_id IS NOT NULL
            AND category_id IS NULL)
        OR
        (type IN ('income', 'expense')
            AND counter_account_id IS NULL)
    ),

    -- Không thể chuyển tiền từ một ví sang chính nó.
    CONSTRAINT transactions_accounts_differ CHECK (
        counter_account_id IS NULL OR counter_account_id <> account_id
    )
);

-- Index chính phục vụ màn hình danh sách giao dịch: lọc theo người dùng,
-- sắp xếp mới nhất trước. Thêm cột id để phân trang bằng con trỏ có thứ
-- tự ổn định khi nhiều giao dịch trùng occurred_at.
CREATE INDEX transactions_user_occurred_idx
    ON transactions (user_id, occurred_at DESC, id DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX transactions_account_idx
    ON transactions (account_id)
    WHERE deleted_at IS NULL;

CREATE INDEX transactions_counter_account_idx
    ON transactions (counter_account_id)
    WHERE deleted_at IS NULL AND counter_account_id IS NOT NULL;

CREATE INDEX transactions_category_idx
    ON transactions (category_id)
    WHERE deleted_at IS NULL AND category_id IS NOT NULL;

CREATE TRIGGER transactions_set_updated_at
    BEFORE UPDATE ON transactions
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transactions;
-- +goose StatementEnd
