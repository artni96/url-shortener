package model

type URLCreateRequest struct {
	URL string `json:"url"`
}

type URLCreateResponse struct {
	ID     string `json:"-"`
	Result string `json:"result"`
}

type URLEntity struct {
	//ID          int    `json:"id"`
	OriginalURL string `json:"url"`
	ShortURL    string `json:"short_url"`
}
