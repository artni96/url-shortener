package urls

import (
	"errors"
	"fmt"
)

var (
	ErrShortURLAlreadyExists    = errors.New("short url already exists")
	ErrOriginalURLAlreadyExists = errors.New("url already exists")
	ErrURLNotFound              = errors.New("url not found")
	ErrURLNotCreated            = errors.New("could not create url")
	ErrURLListIsNotUnique       = errors.New("url list is not unique")
	ErrDuplicatedURL            = errors.New(fmt.Sprintf("duplicated url"))
	ErrUserIsNotAuthor          = errors.New("user is not url author")
)
