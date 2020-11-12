package rfc2822

import (
	"mime"
	"strings"
)

func ParseContentType(s string) (ct ContentType, err error) {
	mdType, params, err := mime.ParseMediaType(s)

	if err != nil {
		return ContentType{}, err
	}

	types := strings.Split(mdType, "/")

	ct.Type = types[0]

	ct.SubType = strings.Join(types[1:], "/")

	ct.Params = params

	return
}

func ParseContentDisposition(s string) (ct ContentDisposition, err error) {
	ct.MediaType, ct.Params, err = mime.ParseMediaType(s)
	if err != nil {
		return ContentDisposition{}, err
	}

	return
}
