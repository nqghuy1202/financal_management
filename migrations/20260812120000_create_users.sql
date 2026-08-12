-- +goose Up
-- +goose StatementBegin

-- citext cho phép so sánh chuỗi không phân biệt hoa thường ngay ở tầng
-- database. Dùng cho email: "Huy@Gmail.com" và "huy@gmail.com" phải là
-- cùng một tài khoản. Nếu để kiểu text, ràng buộc UNIQUE sẽ cho đăng ký
-- trùng chỉ khác hoa thường.
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
    id                UUID        PRIMARY KEY,
    email             CITEXT      NOT NULL,
    password_hash     TEXT        NOT NULL,
    full_name         TEXT        NOT NULL,

    -- ISO 4217. Đây là loại tiền mặc định khi người dùng tạo ví mới.
    base_currency     CHAR(3)     NOT NULL DEFAULT 'VND',

    -- NULL nghĩa là chưa xác thực email. Dùng timestamp thay cho cờ
    -- boolean để biết luôn thời điểm xác thực.
    email_verified_at TIMESTAMPTZ,

    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT users_email_not_blank    CHECK (length(trim(email::text)) > 0),
    CONSTRAINT users_full_name_not_blank CHECK (length(trim(full_name)) > 0),
    -- Chặn ở tầng DB luôn, phòng khi có đường ghi dữ liệu nào bỏ qua
    -- validate ở tầng ứng dụng.
    CONSTRAINT users_base_currency_format CHECK (base_currency ~ '^[A-Z]{3}$')
);

CREATE UNIQUE INDEX users_email_key ON users (email);

-- +goose StatementEnd

-- +goose StatementBegin

-- Trigger giữ updated_at luôn đúng, kể cả khi ai đó UPDATE trực tiếp
-- bằng psql mà quên set cột này. Đặt ở DB thì không có đường nào lách.
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS users_set_updated_at ON users;
DROP TABLE IF EXISTS users;
DROP FUNCTION IF EXISTS set_updated_at();
-- +goose StatementEnd
