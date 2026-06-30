package model

// AuditEntity represents an Audit entity.
// generate:reset
type AuditEntity struct {
	Ts     int64  `json:"ts"`
	Action string `json:"action"`
	UserID int    `json:"user_id,omitempty"`
	URL    string `json:"url"`
}
