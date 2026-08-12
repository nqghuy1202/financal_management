package controller

import (
	"context"
	"net/http"
	"time"

	"financal_management/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Khoá dùng lặp lại trong payload JSON của các endpoint health.
const (
	keyStatus = "status"
	statusUp  = "up"
)

// HealthController phục vụ các endpoint kiểm tra tình trạng ứng dụng.
//
// Nhận thẳng pool và client thay vì đi qua service/repo, vì đây là
// endpoint hạ tầng chứ không phải nghiệp vụ — thêm hai tầng ở giữa chỉ
// làm rối mà không mang lại giá trị.
type HealthController struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthController(db *pgxpool.Pool, rdb *redis.Client) *HealthController {
	return &HealthController{db: db, redis: rdb}
}

// Live trả về 200 ngay khi tiến trình còn sống.
//
// Đây là liveness probe: chỉ nói "tiến trình chưa treo". Không kiểm tra
// dependency ở đây, vì nếu DB sập mà probe này fail thì orchestrator sẽ
// restart ứng dụng một cách vô ích.
func (hc *HealthController) Live(c *gin.Context) {
	response.Success(c, gin.H{keyStatus: "alive"})
}

// Ready kiểm tra ứng dụng có sẵn sàng nhận traffic hay không.
//
// Đây là readiness probe: có ping thật xuống PostgreSQL và Redis. Nếu
// một dependency chết, endpoint trả 503 để load balancer ngừng đẩy
// request sang instance này.
func (hc *HealthController) Ready(c *gin.Context) {
	// Timeout ngắn: probe phải trả lời nhanh, chậm hơn ngưỡng này thì coi
	// như dependency đã có vấn đề.
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	checks := gin.H{}
	healthy := true

	if err := hc.db.Ping(ctx); err != nil {
		checks["postgres"] = gin.H{keyStatus: "down", "error": err.Error()}
		healthy = false
	} else {
		checks["postgres"] = gin.H{keyStatus: statusUp}
	}

	if err := hc.redis.Ping(ctx).Err(); err != nil {
		checks["redis"] = gin.H{keyStatus: "down", "error": err.Error()}
		healthy = false
	} else {
		checks["redis"] = gin.H{keyStatus: statusUp}
	}

	if !healthy {
		c.JSON(http.StatusServiceUnavailable, response.ResponseData{
			Code:    response.CodeDependencyUnavailable,
			Message: response.Message(response.CodeDependencyUnavailable),
			Data:    gin.H{keyStatus: "not_ready", "checks": checks},
		})
		return
	}

	response.Success(c, gin.H{keyStatus: "ready", "checks": checks})
}
