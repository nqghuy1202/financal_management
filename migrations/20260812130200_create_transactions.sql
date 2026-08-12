-- +goose Up
-- +goose StatementBegin

CREATE TABLE transactions (
    id          UUID          PRIMARY KEY,
    user_id     UUID          NOT NULL REFERENCES users (id) ON DELETE RESTRICT,

    -- Nguồn tiền. Cho phép NULL vì người dùng có thể chỉ muốn ghi nhanh
    -- "hôm nay ăn trưa 50k" mà không quan tâm trả bằng gì. Bắt buộc chọn
    -- nguồn sẽ làm việc nhập liệu hằng ngày trở nên phiền.
    account_id  UUID          REFERENCES accounts (id)   ON DELETE RESTRICT,

    -- Danh mục. Cũng cho phép NULL để ghi nhanh; báo cáo gom các khoản
    -- này vào nhóm "Chưa phân loại".
    category_id UUID          REFERENCES categories (id) ON DELETE RESTRICT,

    -- income: khoản thu, expense: khoản chi. Không có loại nào khác —
    -- ứng dụng không hỗ trợ chuyển tiền giữa các nguồn.
    type        TEXT          NOT NULL,

    -- Luôn là số dương. Tiền vào hay ra được suy ra từ cột type, không
    -- dùng dấu âm — nếu dùng dấu, mọi câu tổng hợp đều phải nhớ xử lý
    -- dấu và rất dễ sai.
    amount      NUMERIC(19,4) NOT NULL,

    currency    CHAR(3)       NOT NULL DEFAULT 'VND',

    note        TEXT          NOT NULL DEFAULT '',

    -- Thời điểm tiền thực sự chi ra hoặc thu vào, do người dùng nhập.
    -- Khác created_at là thời điểm bản ghi được tạo trong hệ thống: hôm
    -- nay nhập bữa ăn của hôm qua thì báo cáo phải tính vào hôm qua.
    -- Toàn bộ báo cáo tổng hợp theo cột này.
    occurred_at TIMESTAMPTZ   NOT NULL,

    deleted_at  TIMESTAMPTZ,

    created_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),

    CONSTRAINT transactions_type_valid      CHECK (type IN ('income', 'expense')),
    CONSTRAINT transactions_amount_positive CHECK (amount > 0),
    CONSTRAINT transactions_currency_format CHECK (currency ~ '^[A-Z]{3}$')
);

-- Index chính phục vụ cả màn hình danh sách lẫn mọi báo cáo theo kỳ:
-- lọc theo người dùng rồi lấy khoảng thời gian, mới nhất trước.
-- Thêm cột id để phân trang bằng con trỏ có thứ tự ổn định khi nhiều
-- giao dịch trùng occurred_at.
CREATE INDEX transactions_user_occurred_idx
    ON transactions (user_id, occurred_at DESC, id DESC)
    WHERE deleted_at IS NULL;

-- Báo cáo "chi tiêu theo danh mục" lọc theo người dùng, loại và khoảng
-- thời gian rồi gom nhóm theo danh mục.
CREATE INDEX transactions_user_type_occurred_idx
    ON transactions (user_id, type, occurred_at)
    WHERE deleted_at IS NULL;

CREATE INDEX transactions_category_idx
    ON transactions (category_id)
    WHERE deleted_at IS NULL AND category_id IS NOT NULL;

CREATE INDEX transactions_account_idx
    ON transactions (account_id)
    WHERE deleted_at IS NULL AND account_id IS NOT NULL;

CREATE TRIGGER transactions_set_updated_at
    BEFORE UPDATE ON transactions
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transactions;
-- +goose StatementEnd
