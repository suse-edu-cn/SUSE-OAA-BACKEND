package request

type LoginReq struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}
