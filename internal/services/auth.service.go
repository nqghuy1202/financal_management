package services

import (
	"context"
	"errors"
	"strings"

	"financal_management/internal/pkg/password"
	"financal_management/internal/pkg/response"
	"financal_management/internal/pkg/token"
	"financal_management/internal/repo"
	"financal_management/internal/repo/sqlc"

	"github.com/google/uuid"
)

// AuthService xử lý đăng ký, đăng nhập và quản lý phiên đăng nhập.
type AuthService struct {
	users  repo.UserRepo
	tokens *token.Manager
}

func NewAuthService(users repo.UserRepo, tokens *token.Manager) *AuthService {
	return &AuthService{users: users, tokens: tokens}
}

// RegisterInput là dữ liệu cần để tạo tài khoản mới.
type RegisterInput struct {
	Email    string
	Password string
	FullName string
}

// Session là kết quả của đăng ký, đăng nhập hoặc làm mới token.
type Session struct {
	AccessToken  string
	RefreshToken string
	User         sqlc.User
}

// dummyHash là chuỗi bcrypt của một mật khẩu bất kỳ.
//
// Dùng khi đăng nhập với email không tồn tại: ta vẫn chạy một phép so
// sánh giả để thời gian phản hồi giống hệt trường hợp email có thật.
// Nếu không làm vậy, email không tồn tại sẽ trả lời nhanh hơn hẳn (vì bỏ
// qua bcrypt), và kẻ tấn công chỉ cần đo thời gian là biết email nào đã
// đăng ký trong hệ thống.
// Đây là chuỗi bcrypt thật (cost 12) của một mật khẩu không ai biết, nên
// phép so sánh tốn đúng bằng thời gian so sánh một mật khẩu thật.
const dummyHash = "$2a$12$ii1KS/CFkx/sOl9VxNJX1.GJnFEBQmViOuM1vfa9/NzrOUDbotUaG"

// Register tạo tài khoản mới và đăng nhập luôn.
func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*Session, error) {
	email := normalizeEmail(in.Email)

	exists, err := s.users.EmailExists(ctx, email)
	if err != nil {
		return nil, response.Wrap(response.CodeDatabaseError, err)
	}
	if exists {
		return nil, response.New(response.CodeEmailAlreadyExists)
	}

	hashed, err := password.Hash(in.Password)
	if err != nil {
		return nil, response.Wrap(response.CodeInternalError, err)
	}

	// UUIDv7 có tiền tố là thời gian nên các id sinh liên tiếp nằm gần
	// nhau, giúp index không bị phân mảnh như UUID ngẫu nhiên.
	id, err := uuid.NewV7()
	if err != nil {
		return nil, response.Wrap(response.CodeInternalError, err)
	}

	user, err := s.users.Create(ctx, sqlc.CreateUserParams{
		ID:           id,
		Email:        email,
		PasswordHash: hashed,
		FullName:     strings.TrimSpace(in.FullName),
		BaseCurrency: "VND",
	})
	if err != nil {
		return nil, response.Wrap(response.CodeDatabaseError, err)
	}

	return s.issueSession(ctx, user)
}

// Login kiểm tra thông tin đăng nhập và cấp phiên mới.
func (s *AuthService) Login(ctx context.Context, email, plainPassword string) (*Session, error) {
	user, err := s.users.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, repo.ErrUserNotFound) {
			// So sánh giả để thời gian phản hồi giống trường hợp email có
			// thật — xem chú thích ở dummyHash.
			_ = password.Verify(dummyHash, plainPassword)
			return nil, response.New(response.CodeCredentialsInvalid)
		}
		return nil, response.Wrap(response.CodeDatabaseError, err)
	}

	if err := password.Verify(user.PasswordHash, plainPassword); err != nil {
		if errors.Is(err, password.ErrMismatch) {
			// Cùng một mã lỗi cho "email không tồn tại" và "sai mật khẩu".
			// Nếu tách ra, kẻ tấn công dò được email nào đã đăng ký.
			return nil, response.New(response.CodeCredentialsInvalid)
		}
		return nil, response.Wrap(response.CodeInternalError, err)
	}

	return s.issueSession(ctx, user)
}

// Refresh đổi refresh token lấy cặp token mới.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*Session, error) {
	// ConsumeRefresh xoá token cũ ngay khi đọc, nên mỗi refresh token chỉ
	// dùng được một lần.
	userID, err := s.tokens.ConsumeRefresh(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, token.ErrInvalidToken) {
			return nil, response.New(response.CodeTokenInvalid)
		}
		return nil, response.Wrap(response.CodeInternalError, err)
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repo.ErrUserNotFound) {
			// Token còn hạn nhưng tài khoản đã bị xoá.
			return nil, response.New(response.CodeTokenInvalid)
		}
		return nil, response.Wrap(response.CodeDatabaseError, err)
	}

	return s.issueSession(ctx, user)
}

// Logout thu hồi refresh token.
//
// Access token vẫn còn hiệu lực tới khi hết hạn (tối đa 15 phút) vì JWT
// không thu hồi được — đó là lý do access token được đặt thời hạn ngắn.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if err := s.tokens.RevokeRefresh(ctx, refreshToken); err != nil {
		return response.Wrap(response.CodeInternalError, err)
	}
	return nil
}

// Me trả về thông tin người dùng đang đăng nhập.
func (s *AuthService) Me(ctx context.Context, userID uuid.UUID) (sqlc.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repo.ErrUserNotFound) {
			return sqlc.User{}, response.New(response.CodeNotFound)
		}
		return sqlc.User{}, response.Wrap(response.CodeDatabaseError, err)
	}
	return user, nil
}

// issueSession cấp cặp access token và refresh token cho người dùng.
func (s *AuthService) issueSession(ctx context.Context, user sqlc.User) (*Session, error) {
	accessToken, err := s.tokens.GenerateAccess(user.ID, user.Email)
	if err != nil {
		return nil, response.Wrap(response.CodeInternalError, err)
	}

	refreshToken, err := s.tokens.GenerateRefresh(ctx, user.ID)
	if err != nil {
		return nil, response.Wrap(response.CodeInternalError, err)
	}

	return &Session{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

// normalizeEmail bỏ khoảng trắng thừa và đưa về chữ thường.
//
// Cột email trong database là citext nên đã không phân biệt hoa thường,
// nhưng chuẩn hoá ở đây để dữ liệu lưu xuống luôn đồng nhất.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
