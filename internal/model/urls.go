package model

type URLCreateRequest struct {
	OriginalURL string `json:"url"`
}

type URLCreateResponse struct {
	ID     string `json:"-"`
	Result string `json:"result"`
}

type URLEntity struct {
	OriginalURL string `json:"original_url" db:"original_url"`
	ShortURL    string `json:"short_url" db:"short_url"`
}

type URLBulkCreateRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type URLBulkCreate struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
	OriginalURL   string `json:"original_url"`
}
type URLBulkCreateResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}
