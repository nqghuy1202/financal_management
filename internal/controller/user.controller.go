package controller

import (
	"financal_management/internal/services"
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
