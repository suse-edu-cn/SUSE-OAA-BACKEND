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
}
