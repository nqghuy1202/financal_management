package initialize

import (
	"financal_management/internal/controller"
	"financal_management/internal/middlewares"
	"fmt"

	"github.com/gin-gonic/gin"
)

func AA() gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("Before -> AA")
		c.Next()
		fmt.Println("Alter -> AA")
	}
}

func BB() gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("Before -> BB")
		c.Next()
		fmt.Println("Alter -> BB")
	}
}

func CC(c *gin.Context) {
	fmt.Println("Before -> CC")
	c.Next()
	fmt.Println("Alter -> CC")
}

func InitRouter() *gin.Engine {
	r := gin.Default()
	r.Use(middlewares.AuthenMiddleware(), BB(), CC)
	v1 := r.Group("v1/2026")
	{
		v1.GET("/info", controller.NewUserController().GetResponseStatus)
	}
	// r.GET("/info", controller.NewUserController().GetResponseStatus)
	// r.Run()
	return r
}
