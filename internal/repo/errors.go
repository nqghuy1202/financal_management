package repo

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrDuplicate trả về khi vi phạm ràng buộc duy nhất, ví dụ tạo hai ví
// trùng tên trong cùng một tài khoản.
var ErrDuplicate = errors.New("dữ liệu đã tồn tại")

// pgUniqueViolation là mã lỗi chuẩn SQLSTATE cho vi phạm ràng buộc
// duy nhất. Postgres trả mã này cho mọi UNIQUE index bị vi phạm.
const pgUniqueViolation = "23505"

// isUniqueViolation nhận biết lỗi trùng dữ liệu từ Postgres.
//
// Hàm này nằm ở tầng repo để phần còn lại của ứng dụng không phải biết
// tới mã lỗi của Postgres hay import driver.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation
}
