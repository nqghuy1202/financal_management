package routers

import (
	"github.com/gin-gonic/gin"
)

// registerCategoryRoutes gắn các endpoint quản lý danh mục thu/chi.
func registerCategoryRoutes(rg *gin.RouterGroup, d Deps) {
	categories := rg.Group("/categories")
	categories.Use(d.RequireAuth, d.UserRateLimit)
	{
		categories.GET("", d.Category.List)
		categories.GET("/:id", d.Category.Get)
		categories.POST("", d.Category.Create)
		categories.PUT("/:id", d.Category.Update)
		categories.DELETE("/:id", d.Category.Delete)
	}
}
