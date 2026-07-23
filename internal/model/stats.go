package model

type StatsResponse struct {
	URLs  int64 `json:"urls"`
	Users int64 `json:"users"`
}
