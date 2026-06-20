package model

// generate:reset
type URLCreateRequestWithoutUser struct {
	OriginalURL string `json:"url"`
}

// generate:reset
type URLCreateRequest struct {
	OriginalURL string `json:"url"`
	CreatedBy   int    `json:"created_by"`
}

// generate:reset
type URLCreate struct {
	OriginalURL string `json:"url"`
	ShortURL    string `json:"short_url"`
	CreatedBy   int    `json:"created_by"`
}

// generate:reset
type URLCreateResponse struct {
	ID     string `json:"-"`
	Result string `json:"result"`
}

// generate:reset
type GetByShortURLResponse struct {
	OriginalURL string `json:"original_url" db:"original_url"`
	ShortURL    string `json:"short_url" db:"short_url"`
	IsDeleted   bool   `json:"is_deleted" db:"is_deleted"`
}

// generate:reset
type URLEntity struct {
	OriginalURL string `json:"original_url" db:"original_url"`
	ShortURL    string `json:"short_url" db:"short_url"`
	CreatedBy   int    `json:"created_by" db:"created_by"`
	IsDeleted   bool   `json:"is_deleted" db:"is_deleted"`
}

// generate:reset
type URLListEntity struct {
	OriginalURL string `json:"original_url" db:"original_url"`
	ShortURL    string `json:"short_url" db:"short_url"`
}

// generate:reset
type URLBulkCreateRequestWithoutUser struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// generate:reset
type URLBulkCreateRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
	CreatedBy     int    `json:"created_by"`
}

// generate:reset
type URLBulkCreate struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
	OriginalURL   string `json:"original_url"`
	CreatedBy     int    `json:"created_by"`
}

// generate:reset
type URLBulkCreateResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// generate:reset
type URLNestedData struct {
	OriginalURL string `json:"original_url"`
	CreatedBy   int    `json:"created_by"`
	IsDeleted   bool   `json:"is_deleted"`
}

// generate:reset
type URLUpdateResponse struct {
	OriginalURL string `json:"url"`
}

// generate:reset
type URLDelete struct {
	ShortURL  string `json:"short_url"`
	CreatedBy int    `json:"created_by"`
}
