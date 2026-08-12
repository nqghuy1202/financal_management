package controller

import (
	"financal_management/internal/pkg/response"
	"financal_management/internal/repo/sqlc"
	"financal_management/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CategoryController struct {
	categories *services.CategoryService
}

func NewCategoryController(categories *services.CategoryService) *CategoryController {
	return &CategoryController{categories: categories}
}

// ---------------------------------------------------------------------
// Kiểu dữ liệu vào/ra
// ---------------------------------------------------------------------

type createCategoryRequest struct {
	Name  string `json:"name"  binding:"required,min=1,max=100"`
	Type  string `json:"type"  binding:"required,oneof=income expense"`
	Icon  string `json:"icon"  binding:"max=50"`
	Color string `json:"color" binding:"max=20"`
}

type updateCategoryRequest struct {
	Name  string `json:"name"  binding:"required,min=1,max=100"`
	Icon  string `json:"icon"  binding:"max=50"`
	Color string `json:"color" binding:"max=20"`
}

type categoryResponse struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Type  string    `json:"type"`
	Icon  string    `json:"icon"`
	Color string    `json:"color"`
	// IsSystem cho frontend biết danh mục nào không sửa/xoá được, để ẩn
	// nút tương ứng thay vì để người dùng bấm rồi nhận lỗi.
	IsSystem bool `json:"is_system"`
}

func toCategoryResponse(c sqlc.Category) categoryResponse {
	return categoryResponse{
		ID:       c.ID,
		Name:     c.Name,
		Type:     c.Type,
		Icon:     c.Icon,
		Color:    c.Color,
		IsSystem: c.UserID == nil,
	}
}

// ---------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------

// List liệt kê danh mục. GET /api/v1/categories?type=expense
func (cc *CategoryController) List(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	// Tham số type là tuỳ chọn; để trống thì lấy cả thu lẫn chi.
	categories, err := cc.categories.List(c.Request.Context(), userID, c.Query("type"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	items := make([]categoryResponse, 0, len(categories))
	for _, cat := range categories {
		items = append(items, toCategoryResponse(cat))
	}

	response.Success(c, gin.H{keyItems: items})
}

// Get lấy chi tiết một danh mục. GET /api/v1/categories/:id
func (cc *CategoryController) Get(c *gin.Context) {
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	category, err := cc.categories.Get(c.Request.Context(), id, userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, toCategoryResponse(category))
}

// Create tạo danh mục riêng. POST /api/v1/categories
func (cc *CategoryController) Create(c *gin.Context) {
	var req createCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Wrap(response.CodeValidationFailed, err))
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	category, err := cc.categories.Create(c.Request.Context(), services.CreateCategoryInput{
		UserID: userID,
		Name:   req.Name,
		Type:   req.Type,
		Icon:   req.Icon,
		Color:  req.Color,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, toCategoryResponse(category))
}

// Update sửa danh mục. PUT /api/v1/categories/:id
func (cc *CategoryController) Update(c *gin.Context) {
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req updateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Wrap(response.CodeValidationFailed, err))
		return
	}

	category, err := cc.categories.Update(c.Request.Context(), services.UpdateCategoryInput{
		ID:     id,
		UserID: userID,
		Name:   req.Name,
		Icon:   req.Icon,
		Color:  req.Color,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, toCategoryResponse(category))
}

// Delete xoá danh mục. DELETE /api/v1/categories/:id
func (cc *CategoryController) Delete(c *gin.Context) {
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	if err := cc.categories.Delete(c.Request.Context(), id, userID); err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, gin.H{keyMessage: "Đã xoá danh mục"})
}
