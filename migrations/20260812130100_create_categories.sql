-- +goose Up
-- +goose StatementBegin

CREATE TABLE categories (
    id         UUID        PRIMARY KEY,

    -- NULL = danh mục hệ thống, dùng chung cho mọi người dùng.
    -- Có giá trị = danh mục do người dùng đó tự tạo.
    user_id    UUID        REFERENCES users (id) ON DELETE RESTRICT,

    name       TEXT        NOT NULL,

    -- income: khoản thu, expense: khoản chi.
    -- Một danh mục chỉ thuộc một loại: "Lương" không thể là khoản chi.
    type       TEXT        NOT NULL,

    icon       TEXT        NOT NULL DEFAULT '',
    color      TEXT        NOT NULL DEFAULT '',

    deleted_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT categories_name_not_blank CHECK (length(trim(name)) > 0),
    CONSTRAINT categories_type_valid     CHECK (type IN ('income', 'expense'))
);

-- Cần HAI index riêng thay vì một ràng buộc UNIQUE thông thường.
--
-- Lý do: trong SQL, NULL không bằng NULL. Nếu chỉ viết
-- UNIQUE (user_id, name, type) thì hai danh mục hệ thống cùng tên vẫn
-- lọt qua, vì user_id NULL của chúng không được coi là trùng nhau.
CREATE UNIQUE INDEX categories_user_name_type_key
    ON categories (user_id, lower(name), type)
    WHERE user_id IS NOT NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX categories_system_name_type_key
    ON categories (lower(name), type)
    WHERE user_id IS NULL AND deleted_at IS NULL;

-- Người dùng luôn xem danh mục của mình cùng với danh mục hệ thống, nên
-- index này phục vụ điều kiện `user_id = $1 OR user_id IS NULL`.
CREATE INDEX categories_user_id_idx
    ON categories (user_id)
    WHERE deleted_at IS NULL;

CREATE TRIGGER categories_set_updated_at
    BEFORE UPDATE ON categories
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS categories;
-- +goose StatementEnd
