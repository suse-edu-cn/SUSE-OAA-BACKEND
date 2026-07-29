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

type UserListReq struct {
	Keyword    string `form:"keyword"`
	Department string `form:"department"`
	Role       string `form:"role"`
	Page       int    `form:"page, default:1"`
	PageSize   int    `form:"page_size, default=20"`
}
