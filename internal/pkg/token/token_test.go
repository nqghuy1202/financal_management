package token

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"financal_management/internal/pkg/setting"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func testManager() *Manager {
	return NewManager(setting.JWTSetting{
		Secret:          strings.Repeat("k", 32),
		Issuer:          "fintrack-test",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}, nil) // nil redis: các test dưới đây chỉ động tới access token
}

func TestGenerateVaParseAccess(t *testing.T) {
	m := testManager()
	userID := uuid.New()

	tokenString, err := m.GenerateAccess(userID, "huy@example.com")
	if err != nil {
		t.Fatalf("GenerateAccess lỗi: %v", err)
	}

	claims, err := m.ParseAccess(tokenString)
	if err != nil {
		t.Fatalf("ParseAccess lỗi: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("UserID = %v, mong đợi %v", claims.UserID, userID)
	}
	if claims.Email != "huy@example.com" {
		t.Errorf("Email = %q, mong đợi %q", claims.Email, "huy@example.com")
	}
}

func TestParseAccess_SaiChuKy(t *testing.T) {
	m := testManager()
	tokenString, err := m.GenerateAccess(uuid.New(), "a@b.com")
	if err != nil {
		t.Fatalf("GenerateAccess lỗi: %v", err)
	}

	// Manager khác khoá bí mật thì không được chấp nhận token này.
	other := NewManager(setting.JWTSetting{
		Secret:         strings.Repeat("x", 32),
		Issuer:         "fintrack-test",
		AccessTokenTTL: 15 * time.Minute,
	}, nil)

	if _, err := other.ParseAccess(tokenString); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("token ký bằng khoá khác phải bị từ chối, nhận được: %v", err)
	}
}

func TestParseAccess_HetHan(t *testing.T) {
	// TTL âm để token vừa tạo ra đã hết hạn.
	m := NewManager(setting.JWTSetting{
		Secret:         strings.Repeat("k", 32),
		Issuer:         "fintrack-test",
		AccessTokenTTL: -time.Minute,
	}, nil)

	tokenString, err := m.GenerateAccess(uuid.New(), "a@b.com")
	if err != nil {
		t.Fatalf("GenerateAccess lỗi: %v", err)
	}

	if _, err := m.ParseAccess(tokenString); !errors.Is(err, ErrExpiredToken) {
		t.Errorf("mong đợi ErrExpiredToken, nhận được: %v", err)
	}
}

// Đây là test quan trọng nhất của package này.
//
// Tấn công "alg=none": kẻ tấn công tự tạo một token khai báo thuật toán
// ký là "none" rồi để phần chữ ký trống. Thư viện JWT nào không kiểm tra
// thuật toán sẽ chấp nhận token đó, và kẻ tấn công đăng nhập được thành
// bất kỳ ai mà không cần biết khoá bí mật.
func TestParseAccess_ChanTokenAlgNone(t *testing.T) {
	m := testManager()

	claims := Claims{
		UserID: uuid.New(),
		Email:  "hacker@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "fintrack-test",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("không tạo được token alg=none để thử: %v", err)
	}

	if _, err := m.ParseAccess(unsigned); err == nil {
		t.Fatal("token alg=none phải bị từ chối")
	}
}

// Token do hệ thống khác cấp (khác issuer) phải bị từ chối, kể cả khi
// tình cờ dùng chung khoá bí mật.
func TestParseAccess_SaiIssuer(t *testing.T) {
	other := NewManager(setting.JWTSetting{
		Secret:         strings.Repeat("k", 32),
		Issuer:         "he-thong-khac",
		AccessTokenTTL: 15 * time.Minute,
	}, nil)

	tokenString, err := other.GenerateAccess(uuid.New(), "a@b.com")
	if err != nil {
		t.Fatalf("GenerateAccess lỗi: %v", err)
	}

	if _, err := testManager().ParseAccess(tokenString); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("token khác issuer phải bị từ chối, nhận được: %v", err)
	}
}

func TestParseAccess_ChuoiRac(t *testing.T) {
	m := testManager()
	for _, s := range []string{"", "abc", "a.b.c", "Bearer xyz"} {
		if _, err := m.ParseAccess(s); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("ParseAccess(%q) phải trả ErrInvalidToken, nhận được: %v", s, err)
		}
	}
}

func TestGenerateAccess_KhongLoMatKhauTrongToken(t *testing.T) {
	m := testManager()
	tokenString, err := m.GenerateAccess(uuid.New(), "huy@example.com")
	if err != nil {
		t.Fatalf("GenerateAccess lỗi: %v", err)
	}

	// Phần thân JWT chỉ được mã hoá base64 chứ không mã hoá kín, nên phải
	// chắc chắn không có gì nhạy cảm lọt vào.
	if strings.Contains(tokenString, "password") || strings.Contains(tokenString, "hash") {
		t.Error("access token chứa dữ liệu nhạy cảm")
	}
}

// Refresh token phải ngẫu nhiên, không đoán được.
func TestGenerateRefresh_KhongTrungNhau(t *testing.T) {
	m := newRedisManager(t)

	ctx := context.Background()
	userID := uuid.New()

	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		tok, err := m.GenerateRefresh(ctx, userID)
		if err != nil {
			t.Fatalf("GenerateRefresh lỗi: %v", err)
		}
		if seen[tok] {
			t.Fatal("sinh ra hai refresh token giống nhau")
		}
		seen[tok] = true
	}
}

func TestConsumeRefresh_ChiDungDuocMotLan(t *testing.T) {
	m := newRedisManager(t)
	ctx := context.Background()
	userID := uuid.New()

	tok, err := m.GenerateRefresh(ctx, userID)
	if err != nil {
		t.Fatalf("GenerateRefresh lỗi: %v", err)
	}

	got, err := m.ConsumeRefresh(ctx, tok)
	if err != nil {
		t.Fatalf("lần dùng đầu tiên phải thành công, lỗi: %v", err)
	}
	if got != userID {
		t.Errorf("UserID = %v, mong đợi %v", got, userID)
	}

	// Dùng lại chính token đó phải bị từ chối.
	if _, err := m.ConsumeRefresh(ctx, tok); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("dùng lại refresh token phải bị từ chối, nhận được: %v", err)
	}
}

func TestConsumeRefresh_TokenLa(t *testing.T) {
	m := newRedisManager(t)

	if _, err := m.ConsumeRefresh(context.Background(), "token-khong-ton-tai"); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("mong đợi ErrInvalidToken, nhận được: %v", err)
	}
}

func TestRevokeRefresh(t *testing.T) {
	m := newRedisManager(t)
	ctx := context.Background()

	tok, err := m.GenerateRefresh(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GenerateRefresh lỗi: %v", err)
	}

	if err := m.RevokeRefresh(ctx, tok); err != nil {
		t.Fatalf("RevokeRefresh lỗi: %v", err)
	}

	if _, err := m.ConsumeRefresh(ctx, tok); !errors.Is(err, ErrInvalidToken) {
		t.Error("token đã thu hồi vẫn dùng được")
	}
}
