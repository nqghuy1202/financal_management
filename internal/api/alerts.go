package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// DismissAlert marks one (categoryId, threshold) alert dismissed for the
// current cycle. The cycle is always recomputed server-side from the
// caller's own settings (never client-supplied), and the underlying
// Dismiss is idempotent, so a repeat click or a stale/unknown alert both
// return 200 rather than 404/500.
func (h *Handler) DismissAlert(c *gin.Context) {
	categoryID := c.Param("categoryId")
	threshold, err := strconv.Atoi(c.Param("threshold"))
	if err != nil || (threshold != 70 && threshold != 90 && threshold != 100) {
		fail(c, http.StatusBadRequest, 40090, "Ngưỡng không hợp lệ")
		return
	}

	ctx := c.Request.Context()
	uid := userIDFrom(c)

	owns, err := h.categories.Owns(ctx, uid, categoryID)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50090, "Không thể đóng cảnh báo")
		return
	}
	if !owns {
		fail(c, http.StatusBadRequest, 40091, "Danh mục không tồn tại")
		return
	}

	settings, err := h.settings.Get(ctx, uid)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50090, "Không thể đóng cảnh báo")
		return
	}
	cycleStart, _ := CycleWindow(settings.CycleStartDay, time.Now())

	if err := NewAlertStateRepo(h.db).Dismiss(ctx, uid, categoryID, cycleStart, threshold); err != nil {
		fail(c, http.StatusInternalServerError, 50090, "Không thể đóng cảnh báo")
		return
	}
	ok(c, gin.H{"dismissed": true})
}
