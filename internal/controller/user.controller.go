package controller

import (
	"financal_management/internal/pkg/response"
	"financal_management/internal/services"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	userService *services.UserService
}

// *UserController: con trỏ
// &UserController: địa chỉ con trỏ

func NewUserController() *UserController {
	return &UserController{
		userService: services.NewUserService(),
	}
}

// controller -> service -> repo -> model -> database

// user controller -> uc
func (uc *UserController) GetUserByID() string {
	return uc.userService.GetInfoUserSerivce()
}

func (uc *UserController) GetResponseStatus(c *gin.Context) {
	// c.JSON(http.StatusOK, response.ResponseData{
	// 	Code:    20001,
	// 	Message: "Success",
	// 	Data:    []string{"Gia Huy", "12/02/2003", "TP.Hồ Chí Minh"},
	// })

	// response.SuccessResponse(c, 20001, []string{"Gia Huy", "12/02/2003", "TP.Hồ Chí Minh"})

	response.ErrResponse(c, 20003, "No need!!!")
}
