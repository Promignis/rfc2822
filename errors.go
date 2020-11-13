package rfc2822

import (
	"errors"
)

var errMaxLineLength = errors.New("Reached maximum read limit for a line")
var errMaxHeaderLines = errors.New("Reached maximum limit for number of header lines")
