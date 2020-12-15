package mime

var isTokenTable = [127]bool{
	'!':  true,
	'#':  true,
	'$':  true,
	'%':  true,
	'&':  true,
	'\'': true,
	'*':  true,
	'+':  true,
	'-':  true,
	'.':  true,
	'0':  true,
	'1':  true,
	'2':  true,
	'3':  true,
	'4':  true,
	'5':  true,
	'6':  true,
	'7':  true,
	'8':  true,
	'9':  true,
	'A':  true,
	'B':  true,
	'C':  true,
	'D':  true,
	'E':  true,
	'F':  true,
	'G':  true,
	'H':  true,
	'I':  true,
	'J':  true,
	'K':  true,
	'L':  true,
	'M':  true,
	'N':  true,
	'O':  true,
	'P':  true,
	'Q':  true,
	'R':  true,
	'S':  true,
	'T':  true,
	'U':  true,
	'W':  true,
	'V':  true,
	'X':  true,
	'Y':  true,
	'Z':  true,
	'^':  true,
	'_':  true,
	'`':  true,
	'a':  true,
	'b':  true,
	'c':  true,
	'd':  true,
	'e':  true,
	'f':  true,
	'g':  true,
	'h':  true,
	'i':  true,
	'j':  true,
	'k':  true,
	'l':  true,
	'm':  true,
	'n':  true,
	'o':  true,
	'p':  true,
	'q':  true,
	'r':  true,
	's':  true,
	't':  true,
	'u':  true,
	'v':  true,
	'w':  true,
	'x':  true,
	'y':  true,
	'z':  true,
	'|':  true,
	'~':  true,
}

var commonHeader map[string]string

var commonHeaderList = []string{
	"Accept",
	"Accept-Charset",
	"Accept-Encoding",
	"Accept-Language",
	"Accept-Ranges",
	"Cache-Control",
	"Cc",
	"Connection",
	"Content-Id",
	"Content-Language",
	"Content-Length",
	"Content-Transfer-Encoding",
	"Content-Type",
	"Cookie",
	"Date",
	"Dkim-Signature",
	"Etag",
	"Expires",
	"From",
	"Host",
	"If-Modified-Since",
	"If-None-Match",
	"In-Reply-To",
	"Last-Modified",
	"Location",
	"Message-Id",
	"Mime-Version",
	"Pragma",
	"Received",
	"Return-Path",
	"Server",
	"Set-Cookie",
	"Subject",
	"To",
	"User-Agent",
	"Via",
	"X-Forwarded-For",
	"X-Imforwards",
	"X-Powered-By",
}

func init() {
	commonHeader = make(map[string]string)
	for _, v := range commonHeaderList {
		commonHeader[v] = v
	}
}

// validHeaderFieldByte reports whether b is a valid byte in a header
// field name. RFC 7230 says:
//   header-field   = field-name ":" OWS field-value OWS
//   field-name     = token
//   tchar = "!" / "#" / "$" / "%" / "&" / "'" / "*" / "+" / "-" / "." /
//           "^" / "_" / "`" / "|" / "~" / DIGIT / ALPHA
//   token = 1*tchar
func validHeaderFieldByte(b byte) bool {
	return int(b) < len(isTokenTable) && isTokenTable[b]
}

// Helpers
// Refer https://golang.org/src/net/textproto/reader.go #596
// eg. canonical key for "accept-encoding" is "Accept-Encoding" as per rfc5322
func canonicalMIMEHeaderKey(s string) string {
	// Quick check for canonical encoding.
	upper := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !validHeaderFieldByte(c) {
			return s
		}
		if upper && 'a' <= c && c <= 'z' {
			return _canonicalMIMEHeaderKey([]byte(s))
		}
		if !upper && 'A' <= c && c <= 'Z' {
			return _canonicalMIMEHeaderKey([]byte(s))
		}
		upper = c == '-'
	}
	return s
}

const toLower = 'a' - 'A'

func _canonicalMIMEHeaderKey(a []byte) string {
	// See if a looks like a header key. If not, return it unchanged.
	for _, c := range a {
		if validHeaderFieldByte(c) {
			continue
		}
		// Don't canonicalize.
		return string(a)
	}

	upper := true
	for i, c := range a {
		// Canonicalize: first letter upper case
		// and upper case after each dash.
		// (Host, User-Agent, If-Modified-Since).
		// MIME headers are ASCII only, so no Unicode issues.
		if upper && 'a' <= c && c <= 'z' {
			c -= toLower
		} else if !upper && 'A' <= c && c <= 'Z' {
			c += toLower
		}
		a[i] = c
		upper = c == '-' // for next time
	}
	// The compiler recognizes m[string(byteSlice)] as a special
	// case, so a copy of a's bytes into a new string does not
	// happen in this map lookup:
	if v := commonHeader[string(a)]; v != "" {
		return v
	}
	return string(a)
}
