package mime

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/quotedprintable"
	"net/mail"
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

/*
	Note:
	The MIME specifications specify that the proper method for encoding Content-Type and Content-Disposition parameter values is the method described in rfc2231.
	However, it is common for some older email clients to improperly encode using the method described in rfc2047 instead.
	mime.ParseMediaType takes care of rfc2231 specs of encoded parameter types and decode it to utf-8
	but it will not handle rfc2047 type encoding and will return an error

	eg, this works
	name*0*=utf-8''%D0%AD%D1%82%D0%BE%20%D1%80%D1%83%D1%81%D1%81%D0%BA%D0%BE;\n\tname*1*=%D0%B5%20%D0%B8%D0%BC%D1%8F%20%D1%84%D0%B0%D0%B9%D0%BB%D0%B0.txt"
	but this does not
	name=\"=?utf-8?b?0K3RgtC+INGA0YPRgdGB0LrQvtC1INC40LzRjyDRhNCw0LnQu9CwLnR4?=\n\t=?utf-8?q?t?=\"

	// TODO: Try to catch exact parsing errors, like in this case say something like
	"badly encoded content disposition param" instead of a genereic "bad mime"
*/
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

func parseAddress(headerVal string) ([]*mail.Address, error) {
	decodedAddr := decodeToUTF8Base64Header(headerVal)
	ret, err := mail.ParseAddressList(decodedAddr)
	if err != nil {
		switch err.Error() {
		case "mail: expected comma":
			return mail.ParseAddressList(ensureCommaDelimitedAddresses(decodedAddr))
		case "mail: no address":
			return nil, mail.ErrHeaderNotPresent
		}
		return nil, err
	}
	return ret, nil
}

func IsInternational(val string) bool {
	for i := 0; i < len(val); i++ {
		if val[i] > 127 {
			return true
		}
	}

	return false
}

// Used by AddressList to ensure that address lists are properly delimited
func ensureCommaDelimitedAddresses(s string) string {
	// This normalizes the whitespace, but may interfere with CFWS (comments with folding whitespace)
	// RFC-5322 3.4.0:
	//      because some legacy implementations interpret the comment,
	//      comments generally SHOULD NOT be used in address fields
	//      to avoid confusing such implementations.
	s = strings.Join(strings.Fields(s), " ")

	inQuotes := false
	inDomain := false
	escapeSequence := false
	sb := strings.Builder{}
	for _, r := range s {
		if escapeSequence {
			escapeSequence = false
			sb.WriteRune(r)
			continue
		}
		if r == '"' {
			inQuotes = !inQuotes
			sb.WriteRune(r)
			continue
		}
		if inQuotes {
			if r == '\\' {
				escapeSequence = true
				sb.WriteRune(r)
				continue
			}
		} else {
			if r == '@' {
				inDomain = true
				sb.WriteRune(r)
				continue
			}
			if inDomain {
				if r == ';' {
					sb.WriteRune(r)
					break
				}
				if r == ',' {
					inDomain = false
					sb.WriteRune(r)
					continue
				}
				if r == ' ' {
					inDomain = false
					sb.WriteRune(',')
					sb.WriteRune(r)
					continue
				}
			}
		}
		sb.WriteRune(r)
	}
	return sb.String()
}
