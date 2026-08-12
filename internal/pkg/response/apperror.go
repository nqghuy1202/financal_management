package response

import (
	"errors"
	"fmt"
)

// AppError là kiểu lỗi dùng xuyên suốt các tầng service và repo.
//
// Ý tưởng: tầng dưới trả về AppError mang mã lỗi nghiệp vụ, middleware
// xử lý lỗi ở tầng trên chỉ việc đọc mã đó ra để dựng response. Nhờ vậy
// controller không phải tự map lỗi sang HTTP status ở từng handler.
//
// Trường cause giữ lỗi gốc để ghi log phục vụ debug, nhưng không bao giờ
// được trả về cho client — tránh lộ thông tin nội bộ như câu SQL hay tên bảng.
type AppError struct {
	Code    int
	Message string
	cause   error
}

func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap cho phép dùng errors.Is và errors.As với lỗi gốc.
func (e *AppError) Unwrap() error { return e.cause }

// Cause trả về lỗi gốc đã được bọc, dùng để ghi log.
func (e *AppError) Cause() error { return e.cause }

// New tạo lỗi với thông báo mặc định của mã.
func New(code int) *AppError {
	return &AppError{Code: code, Message: Message(code)}
}

// Newf tạo lỗi với thông báo tự đặt.
func Newf(code int, format string, args ...any) *AppError {
	return &AppError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// Wrap bọc một lỗi kỹ thuật thành AppError mang mã nghiệp vụ.
// Trả về nil nếu err là nil, để có thể viết `return response.Wrap(code, err)`
// ở cuối hàm mà không cần kiểm tra trước.
func Wrap(code int, err error) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{Code: code, Message: Message(code), cause: err}
}

// Wrapf bọc lỗi kèm thông báo tự đặt.
func Wrapf(code int, err error, format string, args ...any) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{Code: code, Message: fmt.Sprintf(format, args...), cause: err}
}

// AsAppError trích AppError ra khỏi chuỗi lỗi. Nếu err không phải AppError
// thì quy về lỗi hệ thống — mọi lỗi lạ đều được che đi khỏi client.
func AsAppError(err error) *AppError {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return &AppError{Code: CodeInternalError, Message: Message(CodeInternalError), cause: err}
}
