package mime

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/quotedprintable"
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

// This is the exact clone of bufio.ReadBytes method but with an upper limit
// bufio.ReadBytes keeps reading till it finds the delim
// but it could be used to feed the parser very long bad data
func readBytesWithLimit(r *bufio.Reader, delim byte, limit int) ([]byte, error) {
	var frag []byte
	var full [][]byte
	var err error
	n := 0
	for {
		var e error

		if n >= limit {
			// Limit reached, don't read anymore
			err = errMaxLineLength
			break
		}

		frag, e = r.ReadSlice(delim)

		if e == nil { // got final fragment
			break
		}

		if e != bufio.ErrBufferFull { // unexpected error
			err = e
			break
		}

		// Make a copy of the buffer.
		buf := make([]byte, len(frag))
		copy(buf, frag)
		full = append(full, buf)
		n += len(buf)
	}

	if n >= limit {
		buf := make([]byte, n)
		n = 0
		// Copy full pieces and fragment in.
		for i := range full {
			n += copy(buf[n:], full[i])
		}
		return buf, err
	}

	n += len(frag)

	// Allocate new buffer to hold the full pieces and the fragment.
	buf := make([]byte, n)
	n = 0
	// Copy full pieces and fragment in.
	for i := range full {
		n += copy(buf[n:], full[i])
	}
	copy(buf[n:], frag)
	return buf, err
}

func validHeaderKeyByte(b byte) bool {
	c := int(b)
	return c >= 33 && c <= 126 && c != ':'
}

func encodingReader(enc string, r io.Reader) (io.Reader, error) {
	var dec io.Reader
	switch strings.ToLower(enc) {
	case "quoted-printable":
		dec = quotedprintable.NewReader(r)
	case "base64":
		dec = base64.NewDecoder(base64.StdEncoding, r)
	case "7bit", "8bit", "binary", "":
		dec = r
	default:
		return nil, fmt.Errorf("unhandled encoding %q", enc)
	}
	return dec, nil
}
