package routers

import (
	"github.com/gin-gonic/gin"
)

// registerTransactionRoutes gắn các endpoint ghi nhận thu chi.
func registerTransactionRoutes(rg *gin.RouterGroup, d Deps) {
	transactions := rg.Group("/transactions")
	transactions.Use(d.RequireAuth, d.UserRateLimit)
	{
		transactions.POST("", d.Transaction.Create)
		transactions.GET("", d.Transaction.List)
		transactions.GET("/:id", d.Transaction.Get)
		transactions.PUT("/:id", d.Transaction.Update)
		transactions.DELETE("/:id", d.Transaction.Delete)
	}
}
