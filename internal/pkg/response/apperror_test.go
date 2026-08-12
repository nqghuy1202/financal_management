package response

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(CodeNotFound)

	if err.Code != CodeNotFound {
		t.Errorf("Code = %d, mong đợi %d", err.Code, CodeNotFound)
	}
	if err.Message != Message(CodeNotFound) {
		t.Errorf("Message = %q, mong đợi thông báo mặc định của mã", err.Message)
	}
	if err.Cause() != nil {
		t.Errorf("Cause() = %v, mong đợi nil", err.Cause())
	}
}

// Wrap trả về nil khi lỗi đầu vào là nil, để nơi gọi viết được
// `return response.Wrap(code, err)` mà không cần kiểm tra trước.
func TestWrap_NilTraVeNil(t *testing.T) {
	if got := Wrap(CodeDatabaseError, nil); got != nil {
		t.Errorf("Wrap(code, nil) = %v, mong đợi nil", got)
	}
}

// errors.Is phải xuyên qua được AppError để nơi gọi kiểm tra được lỗi gốc.
func TestWrap_GiuLoiGoc(t *testing.T) {
	sentinel := errors.New("connection refused")
	wrapped := Wrap(CodeDatabaseError, sentinel)

	if !errors.Is(wrapped, sentinel) {
		t.Error("errors.Is không tìm thấy lỗi gốc bên trong AppError")
	}
	if wrapped.Cause() != sentinel {
		t.Errorf("Cause() = %v, mong đợi %v", wrapped.Cause(), sentinel)
	}
}

func TestAsAppError(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		wantCode int
		wantNil  bool
	}{
		{
			name:    "nil trả về nil",
			input:   nil,
			wantNil: true,
		},
		{
			name:     "AppError trực tiếp giữ nguyên mã",
			input:    New(CodeEmailAlreadyExists),
			wantCode: CodeEmailAlreadyExists,
		},
		{
			// Đây là điểm mấu chốt: lỗi kỹ thuật vô danh phải được quy về
			// lỗi hệ thống, tuyệt đối không để rò chi tiết ra client.
			name:     "lỗi lạ quy về lỗi hệ thống",
			input:    errors.New("pq: relation \"users\" does not exist"),
			wantCode: CodeInternalError,
		},
		{
			name:     "AppError bị bọc nhiều lớp vẫn trích ra được",
			input:    fmt.Errorf("tầng service: %w", fmt.Errorf("tầng repo: %w", New(CodeNotFound))),
			wantCode: CodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AsAppError(tt.input)

			if tt.wantNil {
				if got != nil {
					t.Fatalf("AsAppError(nil) = %v, mong đợi nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("AsAppError trả về nil, mong đợi một AppError")
			}
			if got.Code != tt.wantCode {
				t.Errorf("Code = %d, mong đợi %d", got.Code, tt.wantCode)
			}
		})
	}
}

// Lỗi lạ bị quy về lỗi hệ thống, nhưng nội dung gốc vẫn phải giữ lại được
// để ghi log phục vụ debug.
func TestAsAppError_GiuLoiGocDeLog(t *testing.T) {
	raw := errors.New("chi tiết nội bộ không được lộ ra ngoài")
	appErr := AsAppError(raw)

	if appErr.Message == raw.Error() {
		t.Error("thông báo trả về client không được là nội dung lỗi gốc")
	}
	if !errors.Is(appErr, raw) {
		t.Error("lỗi gốc phải được giữ lại bên trong để ghi log")
	}
}

func TestHTTPStatus(t *testing.T) {
	tests := []struct {
		code int
		want int
	}{
		{CodeSuccess, http.StatusOK},
		{CodeValidationFailed, http.StatusBadRequest},
		{CodeTokenExpired, http.StatusUnauthorized},
		{CodeForbidden, http.StatusForbidden},
		{CodeNotFound, http.StatusNotFound},
		{CodeEmailAlreadyExists, http.StatusConflict},
		{CodeTooManyRequests, http.StatusTooManyRequests},
		{CodeDatabaseError, http.StatusInternalServerError},
		{CodeDependencyUnavailable, http.StatusServiceUnavailable},
		// Mã không có trong catalog phải rơi về lỗi hệ thống chứ không
		// trả về 0 — nếu không, gin sẽ ghi status 0 và client bị lỗi khó hiểu.
		{99999, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		if got := HTTPStatus(tt.code); got != tt.want {
			t.Errorf("HTTPStatus(%d) = %d, mong đợi %d", tt.code, got, tt.want)
		}
	}
}

// Mọi mã lỗi khai báo trong code.go đều phải có mặt trong catalog, nếu
// không thì thêm hằng số mới mà quên khai báo sẽ âm thầm trả về 500.
func TestCatalog_DayDu(t *testing.T) {
	codes := []int{
		CodeSuccess,
		CodeBadRequest, CodeValidationFailed, CodeInvalidParam,
		CodeUnauthorized, CodeTokenMissing, CodeTokenInvalid, CodeTokenExpired, CodeCredentialsInvalid,
		CodeForbidden, CodeNotFound,
		CodeConflict, CodeEmailAlreadyExists,
		CodeTooManyRequests,
		CodeInternalError, CodeDatabaseError, CodeDependencyUnavailable,
	}

	for _, code := range codes {
		if _, ok := catalog[code]; !ok {
			t.Errorf("mã %d chưa được khai báo trong catalog", code)
		}
	}
}
