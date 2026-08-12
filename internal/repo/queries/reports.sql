-- Toàn bộ báo cáo tổng hợp theo cột occurred_at (thời điểm tiền thực sự
-- thu/chi), KHÔNG theo created_at. Hôm nay nhập bữa ăn của hôm qua thì
-- khoản đó phải nằm trong báo cáo của hôm qua.
--
-- Khoảng thời gian luôn là nửa mở [from, to): lấy >= from và < to. Nhờ
-- vậy hai kỳ liền nhau không bao giờ đếm trùng một giao dịch nằm đúng
-- ranh giới.

-- name: ReportSummary :one
-- Tổng thu, tổng chi và số giao dịch trong một kỳ.
--
-- FILTER (WHERE ...) là cách Postgres gộp nhiều phép tính có điều kiện
-- khác nhau vào MỘT lần quét bảng. Nếu tách thành hai câu SUM riêng thì
-- phải quét bảng hai lần cho cùng một khoảng dữ liệu.
--
-- COALESCE để kỳ không có giao dịch nào trả về 0 thay vì NULL.
SELECT
    COALESCE(SUM(amount) FILTER (WHERE type = 'income'),  0)::NUMERIC(19,4) AS total_income,
    COALESCE(SUM(amount) FILTER (WHERE type = 'expense'), 0)::NUMERIC(19,4) AS total_expense,
    COUNT(*) AS transaction_count
FROM transactions
WHERE user_id = @user_id
  AND deleted_at IS NULL
  AND occurred_at >= @from_time
  AND occurred_at <  @to_time;

-- name: ReportByCategory :many
-- Tổng tiền theo từng danh mục trong một kỳ.
--
-- LEFT JOIN chứ không phải INNER JOIN: giao dịch ghi nhanh không có danh
-- mục vẫn phải xuất hiện trong báo cáo, gom vào nhóm "Chưa phân loại".
-- Dùng INNER JOIN sẽ âm thầm bỏ sót chúng và tổng của báo cáo sẽ nhỏ hơn
-- tổng chi thực tế.
SELECT
    t.category_id                       AS category_id,
    COALESCE(c.name,  '')               AS category_name,
    COALESCE(c.icon,  '')               AS category_icon,
    COALESCE(c.color, '')               AS category_color,
    SUM(t.amount)::NUMERIC(19,4)        AS total_amount,
    COUNT(*)                            AS transaction_count
FROM transactions t
LEFT JOIN categories c ON c.id = t.category_id
WHERE t.user_id = @user_id
  AND t.deleted_at IS NULL
  AND t.type = @type
  AND t.occurred_at >= @from_time
  AND t.occurred_at <  @to_time
GROUP BY t.category_id, c.name, c.icon, c.color
ORDER BY total_amount DESC;

-- name: ReportCashFlow :many
-- Thu và chi gom theo từng tháng.
--
-- Phải đổi về múi giờ địa phương TRƯỚC khi cắt theo tháng. occurred_at
-- lưu theo UTC, nên một khoản chi lúc 06:00 ngày 01/08 giờ Việt Nam
-- tương ứng 23:00 ngày 31/07 UTC — nếu gom theo UTC nó sẽ rơi nhầm vào
-- tháng 7.
--
-- Chỉ trả về tháng CÓ giao dịch; tháng trống được điền ở tầng service để
-- biểu đồ không bị đứt quãng.
SELECT
    date_trunc('month', t.occurred_at AT TIME ZONE sqlc.arg('timezone')::text)::TIMESTAMP AS period,
    COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'income'),  0)::NUMERIC(19,4) AS total_income,
    COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'expense'), 0)::NUMERIC(19,4) AS total_expense
FROM transactions t
WHERE t.user_id = @user_id
  AND t.deleted_at IS NULL
  AND t.occurred_at >= @from_time
  AND t.occurred_at <  @to_time
GROUP BY period
ORDER BY period;
