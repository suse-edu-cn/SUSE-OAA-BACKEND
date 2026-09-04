package request

type RegisterReq struct {
	StudentID string `json:"student_id" binding:"required"`
	Username  string `json:"username" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Email     string `json:"email" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

type LoginReq struct {
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required"`
	Device   string `json:"device" binding:"required"`
}

type RefreshReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
	UserID       uint64 `json:"user_id" binding:"required"`
	Device       string `json:"device" binding:"required"`
}

type LogoutReq struct {
	Device string `json:"device" binding:"required"`
}
type UpdatePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

type SendVerificationCodeReq struct {
	Account string `json:"account" binding:"required" `
	Scene   string `json:"scene" binding:"required" `
}

type ResetPasswordReq struct {
	Account  string `json:"account" binding:"required"`
	Code     string `json:"code" binding:"required"`
	Password string `json:"password" binding:"required"`
}
