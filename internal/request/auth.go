package request

type RegisterReq struct {
	StudentID string `json:"student_id"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

type LoginReq struct {
	Account  string `json:"account"`
	Password string `json:"password"`
	Device   string `json:"device"`
}

type RefreshReq struct {
	RefreshToken string `json:"refresh_token"`
	UserId       uint64 `json:"user_id"`
	Device       string `json:"device"`
}

type LogoutReq struct {
	Device string `json:"device"`
}
type UpdatePasswordReq struct {
	OldPassword  string `json:"old_password"`
	NewPassword1 string `json:"new_password1"`
	NewPassword2 string `json:"new_password2"`
}

type SendVerificationCodeReq struct {
	Type string `json:"type"`
}

type ResetPasswordReq struct {
	Code string `json:"code"`
	Type string `json:"type"`
}
