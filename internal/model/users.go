package model

type UserCreateRequest struct {
	Username         string `json:"username"`
	Password         string `json:"password"`
	RepeatedPassword string `json:"repeated_password"`
}

type UserCreate struct {
	Username       string `json:"username"`
	HashedPassword string `json:"hashed_password"`
}

type UserLogin struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserWithHashedPassword struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}
