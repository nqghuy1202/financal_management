package services

import (
	"financal_management/internal/repo"
)

type UserService struct {
	userRepo *repo.UserRepo
}

func NewUserService() *UserService {
	return &UserService{
		userRepo: repo.NewUserRepo(),
	}
}

// user services -> uc
func (us *UserService) GetInfoUserSerivce() string {
	return us.userRepo.GetInfoUser()
}
