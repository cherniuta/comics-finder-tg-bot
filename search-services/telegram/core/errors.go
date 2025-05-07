package core

import "errors"

var ErrAlreadyExists = errors.New("resource or task already exists")
var ErrUnauthorized = errors.New("authentication failed")
