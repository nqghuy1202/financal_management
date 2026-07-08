package middlewares

import (
	"financal_management/internal/pkg/response"
	"fmt"

	"github.com/gin-gonic/gin"
)

func AuthenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("Get Authorization")
		token := c.GetHeader("Authorization")
		if token != "invalid-token" {
			response.ErrResponse(c, 30001, "")
			c.Abort()
			return
		}
		c.Next()
	}
}
