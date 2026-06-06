package urls

import (
	"errors"
	"fmt"
)

var ErrShortURLAlreadyExists = errors.New("short url already exists")
var ErrOriginalURLAlreadyExists = errors.New("url already exists")
var ErrURLNotFound = errors.New("url not found")
var ErrURLNotCreated = errors.New("could not create url")
var ErrURLListIsNotUnique = errors.New("url list is not unique")
var ErrDuplicatedURL = errors.New(fmt.Sprintf("duplicated url"))
var ErrUserIsNotAuthor = errors.New("user is not url author")
