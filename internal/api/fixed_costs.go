package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fixedCostInput struct {
	Name   string `json:"name"`
	Amount int64  `json:"amount"`
}

func (in fixedCostInput) validate() bool {
	return in.Name != "" && in.Amount > 0
}

func (h *Handler) ListFixedCosts(c *gin.Context) {
	list, err := h.fixedCosts.List(c.Request.Context(), userIDFrom(c))
	if err != nil {
		fail(c, http.StatusInternalServerError, 50050, "Không thể tải chi phí cố định")
		return
	}
	ok(c, list)
}

func (h *Handler) CreateFixedCost(c *gin.Context) {
	var in fixedCostInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, 40050, "Dữ liệu không hợp lệ")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if !in.validate() {
		fail(c, http.StatusBadRequest, 40051, "Tên và số tiền là bắt buộc")
		return
	}

	fc := FixedCost{ID: uuid.NewString(), Name: in.Name, Amount: in.Amount}
	if err := h.fixedCosts.Create(c.Request.Context(), userIDFrom(c), fc); err != nil {
		fail(c, http.StatusInternalServerError, 50051, "Không thể tạo chi phí cố định")
		return
	}
	ok(c, fc)
}

func (h *Handler) UpdateFixedCost(c *gin.Context) {
	var in fixedCostInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, 40050, "Dữ liệu không hợp lệ")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if !in.validate() {
		fail(c, http.StatusBadRequest, 40051, "Tên và số tiền là bắt buộc")
		return
	}

	ctx := c.Request.Context()
	uid := userIDFrom(c)
	id := c.Param("id")

	found, err := h.fixedCosts.Update(ctx, uid, id, in.Name, in.Amount)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50052, "Không thể cập nhật chi phí cố định")
		return
	}
	if !found {
		fail(c, http.StatusNotFound, 40450, "Không tìm thấy chi phí cố định")
		return
	}
	ok(c, FixedCost{ID: id, Name: in.Name, Amount: in.Amount})
}

func (h *Handler) DeleteFixedCost(c *gin.Context) {
	found, err := h.fixedCosts.Delete(c.Request.Context(), userIDFrom(c), c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, 50053, "Không thể xóa chi phí cố định")
		return
	}
	if !found {
		fail(c, http.StatusNotFound, 40450, "Không tìm thấy chi phí cố định")
		return
	}
	ok(c, gin.H{"deleted": true})
}
