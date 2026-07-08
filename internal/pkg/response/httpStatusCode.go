package response

const (
	ErrCodeSuccess      = 20001 // sucess
	ErrCodeParamInvalid = 20003 // Email invalid
	ErrTokenInvalid     = 30001 // Tolen invalid
)

var msg = map[int]string{
	ErrCodeSuccess:      "Success",
	ErrCodeParamInvalid: "Email is invalid",
	ErrTokenInvalid:     "Token is invalid",
}
