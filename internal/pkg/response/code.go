package response

import "net/http"

// Mã lỗi nghiệp vụ của hệ thống.
//
// Quy ước: 2 chữ số đầu tương ứng nhóm HTTP status, 3 chữ số sau phân biệt
// từng lỗi cụ thể. Nhờ vậy client nhìn mã là đoán được loại lỗi, mà vẫn
// phân biệt được chi tiết để hiển thị thông báo phù hợp.
//
//	20xxx — thành công
//	40xxx — request sai
//	41xxx — chưa xác thực
//	43xxx — không đủ quyền
//	44xxx — không tìm thấy
//	45xxx — xung đột dữ liệu
//	42xxx — vượt giới hạn
//	50xxx — lỗi phía server
const (
	CodeSuccess = 20000

	CodeBadRequest       = 40000
	CodeValidationFailed = 40001
	CodeInvalidParam     = 40002

	CodeUnauthorized       = 41000
	CodeTokenMissing       = 41001
	CodeTokenInvalid       = 41002
	CodeTokenExpired       = 41003
	CodeCredentialsInvalid = 41004

	CodeForbidden = 43000

	CodeNotFound = 44000

	CodeConflict           = 45000
	CodeEmailAlreadyExists = 45001

	CodeTooManyRequests = 42900

	CodeInternalError         = 50000
	CodeDatabaseError         = 50001
	CodeDependencyUnavailable = 50003
)

// definition gắn mỗi mã lỗi với HTTP status và thông báo mặc định.
type definition struct {
	httpStatus int
	message    string
}

var catalog = map[int]definition{
	CodeSuccess: {http.StatusOK, "Thành công"},

	CodeBadRequest:       {http.StatusBadRequest, "Yêu cầu không hợp lệ"},
	CodeValidationFailed: {http.StatusBadRequest, "Dữ liệu gửi lên không hợp lệ"},
	CodeInvalidParam:     {http.StatusBadRequest, "Tham số không hợp lệ"},

	CodeUnauthorized:       {http.StatusUnauthorized, "Chưa xác thực"},
	CodeTokenMissing:       {http.StatusUnauthorized, "Thiếu access token"},
	CodeTokenInvalid:       {http.StatusUnauthorized, "Access token không hợp lệ"},
	CodeTokenExpired:       {http.StatusUnauthorized, "Access token đã hết hạn"},
	CodeCredentialsInvalid: {http.StatusUnauthorized, "Email hoặc mật khẩu không đúng"},

	CodeForbidden: {http.StatusForbidden, "Không có quyền truy cập tài nguyên này"},

	CodeNotFound: {http.StatusNotFound, "Không tìm thấy tài nguyên"},

	CodeConflict:           {http.StatusConflict, "Dữ liệu bị xung đột"},
	CodeEmailAlreadyExists: {http.StatusConflict, "Email đã được đăng ký"},

	CodeTooManyRequests: {http.StatusTooManyRequests, "Bạn đã gửi quá nhiều yêu cầu, vui lòng thử lại sau"},

	CodeInternalError:         {http.StatusInternalServerError, "Lỗi hệ thống"},
	CodeDatabaseError:         {http.StatusInternalServerError, "Lỗi truy cập cơ sở dữ liệu"},
	CodeDependencyUnavailable: {http.StatusServiceUnavailable, "Dịch vụ phụ thuộc tạm thời không khả dụng"},
}

// lookup trả về định nghĩa của mã lỗi. Mã lạ được quy về lỗi hệ thống để
// không bao giờ trả về response rỗng nghĩa.
func lookup(code int) definition {
	if d, ok := catalog[code]; ok {
		return d
	}
	return catalog[CodeInternalError]
}

// HTTPStatus trả về HTTP status tương ứng với mã lỗi nghiệp vụ.
func HTTPStatus(code int) int { return lookup(code).httpStatus }

// Message trả về thông báo mặc định của mã lỗi.
func Message(code int) string { return lookup(code).message }
