package core

import "errors"

var ErrNotFound = errors.New("resource is not found")
var ErrBadArguments = errors.New("arguments are not acceptable")
