package controller

import (
	"time"

	"financal_management/internal/model"
	"financal_management/internal/pkg/response"
	"financal_management/internal/repo/sqlc"
	"financal_management/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AccountController struct {
	accounts *services.AccountService
}

func NewAccountController(accounts *services.AccountService) *AccountController {
	return &AccountController{accounts: accounts}
}

// ---------------------------------------------------------------------
// Kiểu dữ liệu vào/ra
// ---------------------------------------------------------------------

type createAccountRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
	Type string `json:"type" binding:"required,oneof=cash bank ewallet credit_card"`
	// Mã tiền tệ 3 chữ cái, ví dụ VND.
	Currency string `json:"currency" binding:"required,len=3,alpha"`
	// Số dư ban đầu, gửi dưới dạng CHUỖI để không mất độ chính xác khi
	// JavaScript đọc số lớn. Bỏ trống thì mặc định bằng 0.
	InitialBalance string `json:"initial_balance" binding:"omitempty,numeric"`
	Icon           string `json:"icon" binding:"max=50"`
}

type updateAccountRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
	Type string `json:"type" binding:"required,oneof=cash bank ewallet credit_card"`
	Icon string `json:"icon" binding:"max=50"`
}

type accountResponse struct {
	ID        uuid.UUID   `json:"id"`
	Name      string      `json:"name"`
	Type      string      `json:"type"`
	Balance   model.Money `json:"balance"`
	Icon      string      `json:"icon"`
	CreatedAt time.Time   `json:"created_at"`
}

func toAccountResponse(a sqlc.Account) (accountResponse, error) {
	balance, err := model.NewMoney(a.Balance, a.Currency)
	if err != nil {
		return accountResponse{}, err
	}
	return accountResponse{
		ID:        a.ID,
		Name:      a.Name,
		Type:      a.Type,
		Balance:   balance,
		Icon:      a.Icon,
		CreatedAt: a.CreatedAt,
	}, nil
}

// ---------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------

// Create tạo ví mới. POST /api/v1/accounts
func (ac *AccountController) Create(c *gin.Context) {
	var req createAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Wrap(response.CodeValidationFailed, err))
		return
	}

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Số dư ban đầu để trống thì hiểu là 0.
	balance := decimal.Zero
	if req.InitialBalance != "" {
		parsed, err := decimal.NewFromString(req.InitialBalance)
		if err != nil {
			_ = c.Error(response.Newf(response.CodeValidationFailed,
				"Số dư ban đầu không hợp lệ: %s", req.InitialBalance))
			return
		}
		balance = parsed
	}

	account, err := ac.accounts.Create(c.Request.Context(), services.CreateAccountInput{
		UserID:         userID,
		Name:           req.Name,
		Type:           req.Type,
		Currency:       req.Currency,
		InitialBalance: balance,
		Icon:           req.Icon,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	respondAccount(c, account)
}

// List liệt kê ví. GET /api/v1/accounts
func (ac *AccountController) List(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	accounts, err := ac.accounts.List(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// Cấp phát sẵn dung lượng vì đã biết trước số phần tử.
	items := make([]accountResponse, 0, len(accounts))
	for _, a := range accounts {
		item, err := toAccountResponse(a)
		if err != nil {
			_ = c.Error(response.Wrap(response.CodeInternalError, err))
			return
		}
		items = append(items, item)
	}

	response.Success(c, gin.H{"items": items})
}

// Get lấy chi tiết một ví. GET /api/v1/accounts/:id
func (ac *AccountController) Get(c *gin.Context) {
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	account, err := ac.accounts.Get(c.Request.Context(), id, userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	respondAccount(c, account)
}

// Update sửa ví. PUT /api/v1/accounts/:id
func (ac *AccountController) Update(c *gin.Context) {
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req updateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Wrap(response.CodeValidationFailed, err))
		return
	}

	account, err := ac.accounts.Update(c.Request.Context(), services.UpdateAccountInput{
		ID:     id,
		UserID: userID,
		Name:   req.Name,
		Type:   req.Type,
		Icon:   req.Icon,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	respondAccount(c, account)
}

// Delete xoá ví. DELETE /api/v1/accounts/:id
func (ac *AccountController) Delete(c *gin.Context) {
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	if err := ac.accounts.Delete(c.Request.Context(), id, userID); err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, gin.H{"message": "Đã xoá ví"})
}

// respond chuyển ví sang dạng trả về rồi ghi response.
func respondAccount(c *gin.Context, account sqlc.Account) {
	item, err := toAccountResponse(account)
	if err != nil {
		_ = c.Error(response.Wrap(response.CodeInternalError, err))
		return
	}
	response.Success(c, item)
}
