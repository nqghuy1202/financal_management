package controller

import (
	"time"

	"financal_management/internal/pkg/response"
	"financal_management/internal/repo/sqlc"
	"financal_management/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	Icon string `json:"icon" binding:"max=50"`
}

type updateAccountRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
	Type string `json:"type" binding:"required,oneof=cash bank ewallet credit_card"`
	Icon string `json:"icon" binding:"max=50"`
}

type accountResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Icon      string    `json:"icon"`
	CreatedAt time.Time `json:"created_at"`
}

func toAccountResponse(a sqlc.Account) accountResponse {
	return accountResponse{
		ID:        a.ID,
		Name:      a.Name,
		Type:      a.Type,
		Icon:      a.Icon,
		CreatedAt: a.CreatedAt,
	}
}

// ---------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------

// Create tạo nguồn tiền mới. POST /api/v1/accounts
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

	account, err := ac.accounts.Create(c.Request.Context(), services.CreateAccountInput{
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

// List liệt kê nguồn tiền. GET /api/v1/accounts
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
		items = append(items, toAccountResponse(a))
	}

	response.Success(c, gin.H{"items": items})
}

// Get lấy chi tiết một nguồn tiền. GET /api/v1/accounts/:id
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

// Update sửa nguồn tiền. PUT /api/v1/accounts/:id
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

// Delete xoá nguồn tiền. DELETE /api/v1/accounts/:id
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

	response.Success(c, gin.H{"message": "Đã xoá nguồn tiền"})
}

func respondAccount(c *gin.Context, account sqlc.Account) {
	response.Success(c, toAccountResponse(account))
}
