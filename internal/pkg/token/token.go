// Package token cấp và kiểm tra token đăng nhập.
//
// Hệ thống dùng hai loại token với vai trò khác hẳn nhau:
//
//   - Access token: JWT, sống ngắn (15 phút). Gửi kèm mọi request. Server
//     chỉ cần kiểm tra chữ ký là biết token hợp lệ, không phải truy vấn
//     database — đó là lý do dùng JWT.
//   - Refresh token: chuỗi ngẫu nhiên, sống dài (7 ngày). Chỉ dùng để
//     xin access token mới. Lưu trong Redis nên thu hồi được ngay lập tức.
//
// Vì sao access token phải sống ngắn: JWT không thu hồi được. Một khi đã
// cấp, nó hợp lệ tới lúc hết hạn dù người dùng đã đăng xuất. Để 15 phút
// nghĩa là thiệt hại tối đa khi token bị lộ chỉ kéo dài 15 phút.
//
// Vì sao refresh token KHÔNG phải JWT: nó cần thu hồi được (đăng xuất,
// đổi mật khẩu). Lưu chuỗi ngẫu nhiên trong Redis thì xoá là xong.
package token

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"financal_management/internal/pkg/setting"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	// ErrInvalidToken: token sai chữ ký, sai định dạng, hoặc không tồn tại.
	ErrInvalidToken = errors.New("token không hợp lệ")
	// ErrExpiredToken: token đúng nhưng đã quá hạn.
	ErrExpiredToken = errors.New("token đã hết hạn")
)

// Claims là nội dung đặt trong access token.
//
// Chỉ nhét những thứ ít thay đổi và không nhạy cảm. Không bao giờ đặt
// mật khẩu hay thông tin cá nhân vào đây: phần thân JWT chỉ được mã hoá
// base64, ai cũng đọc được.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

// Manager cấp và kiểm tra token.
type Manager struct {
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
	redis      *redis.Client
}

func NewManager(cfg setting.JWTSetting, rdb *redis.Client) *Manager {
	return &Manager{
		secret:     []byte(cfg.Secret),
		issuer:     cfg.Issuer,
		accessTTL:  cfg.AccessTokenTTL,
		refreshTTL: cfg.RefreshTokenTTL,
		redis:      rdb,
	}
}

// GenerateAccess tạo access token cho một người dùng.
func (m *Manager) GenerateAccess(userID uuid.UUID, email string) (string, error) {
	now := time.Now()

	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
	}

	// HS256: ký bằng một khoá bí mật duy nhất. Đủ dùng khi chỉ có một
	// service cấp và kiểm tra token. Nếu sau này nhiều service cần tự
	// kiểm tra token thì mới cần chuyển sang RS256 (khoá công khai).
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("ký access token thất bại: %w", err)
	}

	return signed, nil
}

// ParseAccess kiểm tra chữ ký và hạn dùng của access token.
func (m *Manager) ParseAccess(tokenString string) (*Claims, error) {
	claims := &Claims{}

	_, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (any, error) {
			// Bắt buộc kiểm tra thuật toán ký. Nếu bỏ qua bước này, kẻ tấn
			// công có thể gửi token khai báo alg="none" và tự tạo token
			// hợp lệ mà không cần biết khoá bí mật.
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("thuật toán ký không được hỗ trợ: %v", t.Header["alg"])
			}
			return m.secret, nil
		},
		jwt.WithIssuer(m.issuer),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// refreshKey là khoá lưu refresh token trong Redis.
func refreshKey(token string) string { return "refresh:" + token }

// GenerateRefresh tạo refresh token mới và lưu vào Redis.
//
// Redis tự xoá khoá khi hết TTL, nên không cần job dọn token quá hạn.
func (m *Manager) GenerateRefresh(ctx context.Context, userID uuid.UUID) (string, error) {
	// 32 byte ngẫu nhiên từ crypto/rand — không dùng math/rand vì giá trị
	// của nó đoán được nếu biết seed.
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("sinh refresh token thất bại: %w", err)
	}
	tokenString := base64.RawURLEncoding.EncodeToString(buf)

	if err := m.redis.Set(ctx, refreshKey(tokenString), userID.String(), m.refreshTTL).Err(); err != nil {
		return "", fmt.Errorf("lưu refresh token thất bại: %w", err)
	}

	return tokenString, nil
}

// ConsumeRefresh đổi một refresh token lấy id người dùng, đồng thời xoá
// token đó đi.
//
// Xoá ngay khi dùng (rotation) để mỗi refresh token chỉ có tác dụng đúng
// một lần. Nếu token bị lộ mà nạn nhân đã dùng nó rồi thì kẻ trộm không
// dùng lại được nữa.
//
// GETDEL làm cả hai việc trong một lệnh, nên hai request đồng thời không
// thể cùng đổi được một token.
func (m *Manager) ConsumeRefresh(ctx context.Context, tokenString string) (uuid.UUID, error) {
	val, err := m.redis.GetDel(ctx, refreshKey(tokenString)).Result()
	if errors.Is(err, redis.Nil) {
		// Không có trong Redis: token sai, đã hết hạn, hoặc đã dùng rồi.
		return uuid.Nil, ErrInvalidToken
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("đọc refresh token thất bại: %w", err)
	}

	userID, err := uuid.Parse(val)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}

	return userID, nil
}

// RevokeRefresh xoá một refresh token, dùng khi đăng xuất.
func (m *Manager) RevokeRefresh(ctx context.Context, tokenString string) error {
	if err := m.redis.Del(ctx, refreshKey(tokenString)).Err(); err != nil {
		return fmt.Errorf("xoá refresh token thất bại: %w", err)
	}
	return nil
}
