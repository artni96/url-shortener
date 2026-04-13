package model

type UserCreateRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserCreate struct {
	Username       string `json:"username"`
	HashedPassword string `json:"hashed_password"`
}

type UserLogin struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
