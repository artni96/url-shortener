package model

type URLCreateRequest struct {
	URL string `json:"url"`
}

type URLCreateResponse struct {
	ID     string `json:"-"`
	Result string `json:"result"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
