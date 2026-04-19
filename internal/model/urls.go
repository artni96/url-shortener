package model

type URLCreateRequest struct {
	OriginalURL string `json:"url"`
	CreatedBy   int    `json:"created_by"`
}

type URLCreate struct {
	OriginalURL string `json:"url"`
	ShortURL    string `json:"short_url"`
	CreatedBy   int    `json:"created_by"`
}

type URLCreateResponse struct {
	ID     string `json:"-"`
	Result string `json:"result"`
}

type URLEntity struct {
	OriginalURL string `json:"original_url" db:"original_url"`
	ShortURL    string `json:"short_url" db:"short_url"`
	CreatedBy   int    `json:"created_by" db:"created_by"`
	IsDeleted   bool   `json:"is_deleted" db:"is_deleted"`
}

type URLListEntity struct {
	OriginalURL string `json:"original_url" db:"original_url"`
	ShortURL    string `json:"short_url" db:"short_url"`
}

type URLBulkCreateRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
	CreatedBy     int    `json:"created_by"`
}

type URLBulkCreate struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
	OriginalURL   string `json:"original_url"`
	CreatedBy     int    `json:"created_by"`
}

type URLBulkCreateResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type URLNestedData struct {
	OriginalURL string `json:"original_url"`
	CreatedBy   int    `json:"created_by"`
}

type URLUpdateResponse struct {
	OriginalURL string `json:"url"`
}
