package controller

import (
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

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

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

type sessionResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int          `json:"expires_in"` // số giây access token còn hiệu lực
	User         userResponse `json:"user"`
}

func toSessionResponse(s *services.Session) sessionResponse {
	return sessionResponse{
		AccessToken:  s.AccessToken,
		RefreshToken: s.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(global.Config.JWT.AccessTokenTTL.Seconds()),
		User:         toUserResponse(s.User),
	}
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

	response.Success(c, toSessionResponse(session))
}

// Refresh đổi refresh token lấy cặp token mới.
// POST /api/v1/auth/refresh
func (ac *AuthController) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Wrap(response.CodeValidationFailed, err))
		return
	}

	session, err := ac.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, toSessionResponse(session))
}

// Logout thu hồi refresh token. POST /api/v1/auth/logout
func (ac *AuthController) Logout(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Wrap(response.CodeValidationFailed, err))
		return
	}

	if err := ac.auth.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, gin.H{"message": "Đã đăng xuất"})
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
