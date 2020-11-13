package rfc2822

import (
	"errors"
)

var errMaxLineLength = errors.New("Reached maximum read limit")
