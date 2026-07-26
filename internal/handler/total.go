package handler

type TotalHandler struct {
	Auth AuthHandler
	User UserHandler
}

func NewTotalHandler(auth AuthHandler, user UserHandler) TotalHandler {
	return TotalHandler{
		Auth: auth,
		User: user,
	}
}
