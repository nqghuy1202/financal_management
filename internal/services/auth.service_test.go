package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"financal_management/internal/pkg/password"
	"financal_management/internal/pkg/response"
	"financal_management/internal/pkg/setting"
	"financal_management/internal/pkg/token"
	"financal_management/internal/repo"
	"financal_management/internal/repo/sqlc"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ---------------------------------------------------------------------
// Repo giả
// ---------------------------------------------------------------------

// fakeUserRepo lưu người dùng trong map thay vì database.
//
// Viết tay thay vì dùng thư viện sinh mock, vì interface chỉ có 4 phương
// thức — thêm một công cụ nữa vào dự án không đáng.
type fakeUserRepo struct {
	byID    map[uuid.UUID]sqlc.User
	byEmail map[string]sqlc.User

	// forceErr nếu khác nil thì mọi phương thức trả về lỗi này, dùng để
	// kiểm tra nhánh xử lý lỗi database.
	forceErr error
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byID:    make(map[uuid.UUID]sqlc.User),
		byEmail: make(map[string]sqlc.User),
	}
}

func (f *fakeUserRepo) Create(_ context.Context, arg sqlc.CreateUserParams) (sqlc.User, error) {
	if f.forceErr != nil {
		return sqlc.User{}, f.forceErr
	}
	user := sqlc.User{
		ID:           arg.ID,
		Email:        arg.Email,
		PasswordHash: arg.PasswordHash,
		FullName:     arg.FullName,
		BaseCurrency: arg.BaseCurrency,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	f.byID[user.ID] = user
	f.byEmail[user.Email] = user
	return user, nil
}

func (f *fakeUserRepo) GetByID(_ context.Context, id uuid.UUID) (sqlc.User, error) {
	if f.forceErr != nil {
		return sqlc.User{}, f.forceErr
	}
	user, ok := f.byID[id]
	if !ok {
		return sqlc.User{}, repo.ErrUserNotFound
	}
	return user, nil
}

func (f *fakeUserRepo) GetByEmail(_ context.Context, email string) (sqlc.User, error) {
	if f.forceErr != nil {
		return sqlc.User{}, f.forceErr
	}
	user, ok := f.byEmail[email]
	if !ok {
		return sqlc.User{}, repo.ErrUserNotFound
	}
	return user, nil
}

func (f *fakeUserRepo) EmailExists(_ context.Context, email string) (bool, error) {
	if f.forceErr != nil {
		return false, f.forceErr
	}
	_, ok := f.byEmail[email]
	return ok, nil
}

// ---------------------------------------------------------------------
// Hàm dựng test
// ---------------------------------------------------------------------

func newTestService(t *testing.T) (*AuthService, *fakeUserRepo) {
	t.Helper()

	srv := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	tokens := token.NewManager(setting.JWTSetting{
		Secret:          strings.Repeat("k", 32),
		Issuer:          "fintrack-test",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}, rdb)

	users := newFakeUserRepo()
	return NewAuthService(users, tokens), users
}

// wantCode kiểm tra lỗi trả về mang đúng mã nghiệp vụ mong đợi.
func wantCode(t *testing.T, err error, code int) {
	t.Helper()
	if err == nil {
		t.Fatalf("mong đợi lỗi mã %d, nhưng không có lỗi", code)
	}
	appErr := response.AsAppError(err)
	if appErr.Code != code {
		t.Errorf("mã lỗi = %d, mong đợi %d (lỗi: %v)", appErr.Code, code, err)
	}
}

// ---------------------------------------------------------------------
// Register
// ---------------------------------------------------------------------

func TestRegister_ThanhCong(t *testing.T) {
	svc, users := newTestService(t)

	session, err := svc.Register(context.Background(), RegisterInput{
		Email:    "huy@example.com",
		Password: "matkhau12345",
		FullName: "Gia Huy",
	})
	if err != nil {
		t.Fatalf("Register lỗi: %v", err)
	}

	if session.AccessToken == "" || session.RefreshToken == "" {
		t.Error("Register phải trả về cả access token và refresh token")
	}
	if session.User.Email != "huy@example.com" {
		t.Errorf("Email = %q, mong đợi %q", session.User.Email, "huy@example.com")
	}
	if session.User.BaseCurrency != "VND" {
		t.Errorf("BaseCurrency = %q, mong đợi VND", session.User.BaseCurrency)
	}

	// Mật khẩu phải được băm, tuyệt đối không lưu dạng thô.
	stored := users.byEmail["huy@example.com"]
	if stored.PasswordHash == "matkhau12345" {
		t.Fatal("mật khẩu được lưu dạng thô")
	}
	if err := password.Verify(stored.PasswordHash, "matkhau12345"); err != nil {
		t.Errorf("chuỗi băm lưu xuống không kiểm tra được: %v", err)
	}
}

func TestRegister_EmailDaTonTai(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	in := RegisterInput{Email: "huy@example.com", Password: "matkhau12345", FullName: "Huy"}
	if _, err := svc.Register(ctx, in); err != nil {
		t.Fatalf("lần đăng ký đầu phải thành công: %v", err)
	}

	_, err := svc.Register(ctx, in)
	wantCode(t, err, response.CodeEmailAlreadyExists)
}

// Email nhập hoa thường lẫn lộn phải quy về cùng một tài khoản.
func TestRegister_ChuanHoaEmail(t *testing.T) {
	svc, users := newTestService(t)
	ctx := context.Background()

	if _, err := svc.Register(ctx, RegisterInput{
		Email:    "  Huy@Example.COM  ",
		Password: "matkhau12345",
		FullName: "Huy",
	}); err != nil {
		t.Fatalf("Register lỗi: %v", err)
	}

	if _, ok := users.byEmail["huy@example.com"]; !ok {
		t.Fatalf("email chưa được chuẩn hoá, đang lưu: %v", users.byEmail)
	}

	// Đăng ký lại bằng dạng chữ thường phải bị chặn.
	_, err := svc.Register(ctx, RegisterInput{
		Email:    "huy@example.com",
		Password: "matkhau12345",
		FullName: "Huy 2",
	})
	wantCode(t, err, response.CodeEmailAlreadyExists)
}

func TestRegister_LoiDatabase(t *testing.T) {
	svc, users := newTestService(t)
	users.forceErr = errors.New("mất kết nối")

	_, err := svc.Register(context.Background(), RegisterInput{
		Email: "a@b.com", Password: "matkhau12345", FullName: "A",
	})
	wantCode(t, err, response.CodeDatabaseError)
}

// ---------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------

func TestLogin_ThanhCong(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	if _, err := svc.Register(ctx, RegisterInput{
		Email: "huy@example.com", Password: "matkhau12345", FullName: "Huy",
	}); err != nil {
		t.Fatalf("Register lỗi: %v", err)
	}

	session, err := svc.Login(ctx, "huy@example.com", "matkhau12345")
	if err != nil {
		t.Fatalf("Login lỗi: %v", err)
	}
	if session.AccessToken == "" {
		t.Error("Login phải trả về access token")
	}
}

// Đây là test bảo mật quan trọng: sai mật khẩu và email không tồn tại
// phải trả về CÙNG một mã lỗi. Nếu tách ra, kẻ tấn công gửi thử hàng
// loạt email là dò được email nào đã đăng ký trong hệ thống.
func TestLogin_KhongLoEmailNaoDaDangKy(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	if _, err := svc.Register(ctx, RegisterInput{
		Email: "cothat@example.com", Password: "matkhau12345", FullName: "Huy",
	}); err != nil {
		t.Fatalf("Register lỗi: %v", err)
	}

	_, errSaiMatKhau := svc.Login(ctx, "cothat@example.com", "sai-mat-khau")
	_, errKhongCoEmail := svc.Login(ctx, "khongcothat@example.com", "sai-mat-khau")

	codeSaiMatKhau := response.AsAppError(errSaiMatKhau)
	codeKhongCoEmail := response.AsAppError(errKhongCoEmail)

	if codeSaiMatKhau.Code != response.CodeCredentialsInvalid {
		t.Errorf("sai mật khẩu: mã = %d, mong đợi %d", codeSaiMatKhau.Code, response.CodeCredentialsInvalid)
	}
	if codeKhongCoEmail.Code != codeSaiMatKhau.Code {
		t.Errorf("hai trường hợp trả mã khác nhau (%d và %d) — lộ email nào đã đăng ký",
			codeKhongCoEmail.Code, codeSaiMatKhau.Code)
	}
	if codeKhongCoEmail.Message != codeSaiMatKhau.Message {
		t.Error("hai trường hợp trả thông báo khác nhau — lộ email nào đã đăng ký")
	}
}

func TestLogin_EmailKhongPhanBietHoaThuong(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	if _, err := svc.Register(ctx, RegisterInput{
		Email: "huy@example.com", Password: "matkhau12345", FullName: "Huy",
	}); err != nil {
		t.Fatalf("Register lỗi: %v", err)
	}

	if _, err := svc.Login(ctx, "HUY@EXAMPLE.COM", "matkhau12345"); err != nil {
		t.Errorf("đăng nhập bằng email viết hoa phải thành công, lỗi: %v", err)
	}
}

// ---------------------------------------------------------------------
// Refresh / Logout
// ---------------------------------------------------------------------

func TestRefresh_CapTokenMoi(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	first, err := svc.Register(ctx, RegisterInput{
		Email: "huy@example.com", Password: "matkhau12345", FullName: "Huy",
	})
	if err != nil {
		t.Fatalf("Register lỗi: %v", err)
	}

	second, err := svc.Refresh(ctx, first.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh lỗi: %v", err)
	}

	if second.RefreshToken == first.RefreshToken {
		t.Error("Refresh phải cấp refresh token mới, không dùng lại cái cũ")
	}
	if second.User.ID != first.User.ID {
		t.Error("Refresh trả về sai người dùng")
	}
}

// Refresh token cũ phải mất tác dụng ngay sau khi dùng.
func TestRefresh_TokenCuKhongDungLaiDuoc(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	first, err := svc.Register(ctx, RegisterInput{
		Email: "huy@example.com", Password: "matkhau12345", FullName: "Huy",
	})
	if err != nil {
		t.Fatalf("Register lỗi: %v", err)
	}

	if _, err := svc.Refresh(ctx, first.RefreshToken); err != nil {
		t.Fatalf("Refresh lần đầu phải thành công: %v", err)
	}

	_, err = svc.Refresh(ctx, first.RefreshToken)
	wantCode(t, err, response.CodeTokenInvalid)
}

func TestRefresh_TokenLa(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.Refresh(context.Background(), "token-bia-ra")
	wantCode(t, err, response.CodeTokenInvalid)
}

func TestLogout_ThuHoiToken(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	session, err := svc.Register(ctx, RegisterInput{
		Email: "huy@example.com", Password: "matkhau12345", FullName: "Huy",
	})
	if err != nil {
		t.Fatalf("Register lỗi: %v", err)
	}

	if err := svc.Logout(ctx, session.RefreshToken); err != nil {
		t.Fatalf("Logout lỗi: %v", err)
	}

	_, err = svc.Refresh(ctx, session.RefreshToken)
	wantCode(t, err, response.CodeTokenInvalid)
}

// ---------------------------------------------------------------------
// Me
// ---------------------------------------------------------------------

func TestMe_ThanhCong(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	session, err := svc.Register(ctx, RegisterInput{
		Email: "huy@example.com", Password: "matkhau12345", FullName: "Gia Huy",
	})
	if err != nil {
		t.Fatalf("Register lỗi: %v", err)
	}

	user, err := svc.Me(ctx, session.User.ID)
	if err != nil {
		t.Fatalf("Me lỗi: %v", err)
	}
	if user.FullName != "Gia Huy" {
		t.Errorf("FullName = %q, mong đợi %q", user.FullName, "Gia Huy")
	}
}

func TestMe_KhongTonTai(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.Me(context.Background(), uuid.New())
	wantCode(t, err, response.CodeNotFound)
}
