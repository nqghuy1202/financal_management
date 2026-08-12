package controller

import (
	"net/http"
	"time"

	"financal_management/global"
	"financal_management/internal/middlewares"
	"financal_management/internal/pkg/response"
	"financal_management/internal/repo/sqlc"
	"financal_management/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthController struct {
	auth *services.AuthService
}

func NewAuthController(auth *services.AuthService) *AuthController {
	return &AuthController{auth: auth}
}

// ---------------------------------------------------------------------
// Kiểu dữ liệu vào/ra
// ---------------------------------------------------------------------

type registerRequest struct {
	Email    string `json:"email"    binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	FullName string `json:"full_name" binding:"required,min=1,max=100"`
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Không có refreshRequest: refresh token đi trong cookie HttpOnly chứ
// không nằm trong body. Nếu nhận từ body thì frontend phải giữ được token
// trong JavaScript, và cookie HttpOnly mất hết ý nghĩa.

// userResponse là thông tin người dùng trả về cho client.
//
// Đây là struct riêng chứ không dùng thẳng sqlc.User, vì sqlc.User có
// trường PasswordHash. Nếu trả struct đó ra ngoài thì chỉ cần một lần sơ
// ý là lộ toàn bộ mật khẩu đã băm của người dùng.
type userResponse struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	BaseCurrency string    `json:"base_currency"`
	CreatedAt    time.Time `json:"created_at"`
}

func toUserResponse(u sqlc.User) userResponse {
	return userResponse{
		ID:           u.ID,
		Email:        u.Email,
		FullName:     u.FullName,
		BaseCurrency: u.BaseCurrency,
		CreatedAt:    u.CreatedAt,
	}
}

// sessionResponse cố tình KHÔNG chứa refresh token.
//
// Refresh token được gửi về trong cookie HttpOnly. Nếu trả thêm trong
// body thì frontend đọc được nó bằng JavaScript, và toàn bộ lý do dùng
// cookie HttpOnly biến mất.
//
// Access token thì có trong body: frontend giữ nó trong bộ nhớ và đính
// vào header Authorization. Nó chỉ sống 15 phút nên rủi ro thấp hơn hẳn.
type sessionResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int          `json:"expires_in"` // số giây access token còn hiệu lực
	User        userResponse `json:"user"`
}

func toSessionResponse(s *services.Session) sessionResponse {
	return sessionResponse{
		AccessToken: s.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(global.Config.JWT.AccessTokenTTL.Seconds()),
		User:        toUserResponse(s.User),
	}
}

// ---------------------------------------------------------------------
// Cookie chứa refresh token
// ---------------------------------------------------------------------

// sameSiteMode đổi giá trị cấu hình sang kiểu của thư viện chuẩn.
func sameSiteMode(v string) http.SameSite {
	switch v {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

// setRefreshCookie gửi refresh token về trình duyệt dưới dạng cookie
// HttpOnly.
//
// Tham số cuối của SetCookie là httpOnly = true: JavaScript không đọc
// được cookie này, nên XSS không lấy được token mang đi dùng nơi khác.
func setRefreshCookie(c *gin.Context, token string) {
	cfg := global.Config.Cookie
	c.SetSameSite(sameSiteMode(cfg.SameSite))
	c.SetCookie(
		cfg.Name,
		token,
		int(global.Config.JWT.RefreshTokenTTL.Seconds()),
		cfg.Path,
		cfg.Domain,
		cfg.Secure,
		true,
	)
}

// clearRefreshCookie xoá cookie khi đăng xuất.
//
// maxAge âm là cách bảo trình duyệt xoá cookie ngay lập tức.
func clearRefreshCookie(c *gin.Context) {
	cfg := global.Config.Cookie
	c.SetSameSite(sameSiteMode(cfg.SameSite))
	c.SetCookie(cfg.Name, "", -1, cfg.Path, cfg.Domain, cfg.Secure, true)
}

// readRefreshToken đọc refresh token từ cookie.
func readRefreshToken(c *gin.Context) (string, bool) {
	token, err := c.Cookie(global.Config.Cookie.Name)
	if err != nil || token == "" {
		return "", false
	}
	return token, true
}

// ---------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------

// Register tạo tài khoản mới. POST /api/v1/auth/register
func (ac *AuthController) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Wrap(response.CodeValidationFailed, err))
		return
	}

	session, err := ac.auth.Register(c.Request.Context(), services.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	setRefreshCookie(c, session.RefreshToken)
	response.Success(c, toSessionResponse(session))
}

// Login đăng nhập. POST /api/v1/auth/login
func (ac *AuthController) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Wrap(response.CodeValidationFailed, err))
		return
	}

	session, err := ac.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		_ = c.Error(err)
		return
	}

	setRefreshCookie(c, session.RefreshToken)
	response.Success(c, toSessionResponse(session))
}

// Refresh đổi refresh token lấy cặp token mới.
// POST /api/v1/auth/refresh
//
// Không nhận body: token lấy từ cookie do trình duyệt tự đính kèm.
func (ac *AuthController) Refresh(c *gin.Context) {
	refreshToken, ok := readRefreshToken(c)
	if !ok {
		// Ghi rõ thiếu refresh token chứ không dùng thông báo mặc định
		// "Thiếu access token" — frontend cần phân biệt hai trường hợp:
		// thiếu access token thì gọi refresh, thiếu refresh token thì
		// bắt đăng nhập lại.
		_ = c.Error(response.Newf(response.CodeTokenMissing,
			"Thiếu refresh token, vui lòng đăng nhập lại"))
		return
	}

	session, err := ac.auth.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		// Token hỏng hoặc đã dùng rồi: xoá cookie để trình duyệt không
		// gửi lại một giá trị vô dụng ở mọi lần thử sau.
		clearRefreshCookie(c)
		_ = c.Error(err)
		return
	}

	// Refresh token cũ đã bị xoá khỏi Redis, phải thay bằng cái mới.
	setRefreshCookie(c, session.RefreshToken)
	response.Success(c, toSessionResponse(session))
}

// Logout thu hồi refresh token. POST /api/v1/auth/logout
//
// Luôn trả về thành công, kể cả khi không có cookie: người dùng bấm đăng
// xuất hai lần không nên nhận lỗi.
func (ac *AuthController) Logout(c *gin.Context) {
	if refreshToken, ok := readRefreshToken(c); ok {
		if err := ac.auth.Logout(c.Request.Context(), refreshToken); err != nil {
			_ = c.Error(err)
			return
		}
	}

	clearRefreshCookie(c)
	response.Success(c, gin.H{keyMessage: "Đã đăng xuất"})
}

// Me trả về thông tin tài khoản đang đăng nhập.
// GET /api/v1/auth/me — yêu cầu access token.
func (ac *AuthController) Me(c *gin.Context) {
	// Middleware RequireAuth đã đặt giá trị này, nên tới đây chắc chắn có.
	userID, err := uuid.Parse(c.GetString(middlewares.ContextUserIDKey))
	if err != nil {
		_ = c.Error(response.Wrap(response.CodeTokenInvalid, err))
		return
	}

	user, err := ac.auth.Me(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, toUserResponse(user))
}
