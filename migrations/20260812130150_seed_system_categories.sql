-- +goose Up
-- +goose StatementBegin

-- Danh mục hệ thống dùng chung cho mọi người dùng (user_id = NULL).
--
-- Seed bằng migration thay vì sinh lại cho từng người khi đăng ký:
--   - Không nhân bản hàng chục dòng giống nhau cho mỗi tài khoản mới.
--   - Đăng ký chỉ còn một thao tác ghi, không cần transaction nhiều bảng.
--   - Sửa tên hay biểu tượng của một danh mục là sửa một dòng duy nhất.
--
-- Người dùng vẫn tạo được danh mục riêng; khi đó user_id có giá trị.
--
-- Id được ghi cố định để mọi môi trường có cùng id, tiện cho việc gắn
-- biểu tượng ở frontend và cho dữ liệu mẫu.
INSERT INTO categories (id, user_id, name, type, icon, color) VALUES
    -- Khoản chi
    ('019f0000-0000-7000-8000-000000000001', NULL, 'Ăn uống',              'expense', '🍜', '#EF4444'),
    ('019f0000-0000-7000-8000-000000000002', NULL, 'Đi lại',               'expense', '🚌', '#F97316'),
    ('019f0000-0000-7000-8000-000000000003', NULL, 'Nhà ở',                'expense', '🏠', '#F59E0B'),
    ('019f0000-0000-7000-8000-000000000004', NULL, 'Hoá đơn & tiện ích',   'expense', '💡', '#EAB308'),
    ('019f0000-0000-7000-8000-000000000005', NULL, 'Mua sắm',              'expense', '🛍️', '#84CC16'),
    ('019f0000-0000-7000-8000-000000000006', NULL, 'Giải trí',             'expense', '🎬', '#22C55E'),
    ('019f0000-0000-7000-8000-000000000007', NULL, 'Sức khoẻ',             'expense', '💊', '#14B8A6'),
    ('019f0000-0000-7000-8000-000000000008', NULL, 'Giáo dục',             'expense', '📚', '#06B6D4'),
    ('019f0000-0000-7000-8000-000000000009', NULL, 'Quà tặng & từ thiện',  'expense', '🎁', '#8B5CF6'),
    ('019f0000-0000-7000-8000-00000000000a', NULL, 'Chi phí khác',         'expense', '📦', '#6B7280'),

    -- Khoản thu
    ('019f0000-0000-7000-8000-00000000000b', NULL, 'Lương',                'income',  '💰', '#10B981'),
    ('019f0000-0000-7000-8000-00000000000c', NULL, 'Thưởng',               'income',  '🏆', '#059669'),
    ('019f0000-0000-7000-8000-00000000000d', NULL, 'Kinh doanh',           'income',  '🏪', '#0D9488'),
    ('019f0000-0000-7000-8000-00000000000e', NULL, 'Đầu tư',               'income',  '📈', '#0891B2'),
    ('019f0000-0000-7000-8000-00000000000f', NULL, 'Được tặng',            'income',  '🎉', '#7C3AED'),
    ('019f0000-0000-7000-8000-000000000010', NULL, 'Thu nhập khác',        'income',  '💵', '#6B7280')
ON CONFLICT (id) DO NOTHING;

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DELETE FROM categories WHERE user_id IS NULL;
-- +goose StatementEnd
