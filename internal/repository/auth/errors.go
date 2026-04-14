package auth

import "errors"

var ErrUserAlreadyExists = errors.New("user already exists")
var ErrUserNotFound = errors.New("user not found")
var ErrUserNotCreated = errors.New("user not created")
var ErrFailedToCreateNewUser = errors.New("failed to create new user")
