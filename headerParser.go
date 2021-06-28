package rfc2822

import (
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"
)

type headerParser struct {
	s string
}

func (p *headerParser) len() int {
	return len(p.s)
}

func (p *headerParser) empty() bool {
	return p.len() == 0
}

func (p *headerParser) peek() byte {
	return p.s[0]
}

func (p *headerParser) consume(c byte) bool {
	if p.empty() || p.peek() != c {
		return false
	}
	p.s = p.s[1:]
	return true
}

// skipSpace skips the leading space and tab characters.
func (p *headerParser) skipSpace() {
	p.s = strings.TrimLeft(p.s, " \t")
}

func (p *headerParser) skipCFWS() bool {
	p.skipSpace()

	for {
		if !p.consume('(') {
			break
		}

		if _, ok := p.consumeComment(); !ok {
			return false
		}

		p.skipSpace()
	}

	return true
}

func (p *headerParser) consumeComment() (string, bool) {
	// '(' already consumed.
	depth := 1

	var comment string
	for {
		if p.empty() || depth == 0 {
			break
		}

		if p.peek() == '\\' && p.len() > 1 {
			p.s = p.s[1:]
		} else if p.peek() == '(' {
			depth++
		} else if p.peek() == ')' {
			depth--
		}

		if depth > 0 {
			comment += p.s[:1]
		}

		p.s = p.s[1:]
	}

	return comment, depth == 0
}

func (p *headerParser) consumeAtomText(dot, lineant, at bool) (string, error) {
	i := 0
	for {
		r, size := utf8.DecodeRuneInString(p.s[i:])
		if size == 1 && r == utf8.RuneError {
			return "", fmt.Errorf("invalid UTF-8 in atom-text: %q", p.s)
		} else if size == 0 || !isAtext(r, dot, lineant, at) {
			break
		}
		i += size
	}
	if i == 0 {
		return "", fmt.Errorf("Err Empty string")
	}

	var atom string
	atom, p.s = p.s[:i], p.s[i:]
	return atom, nil
}

func isAtext(r rune, dot, lineant, at bool) bool {
	switch r {
	case '.':
		return dot
	// RFC 5322 3.2.3 specials
	case '(', ')', '[', ']', ';', '\\', ',':
		return lineant
	case '@':
		return at
	case '<', '>', '"', ':':
		return false
	}
	return isVchar(r)
}

// isVchar reports whether r is an RFC 5322 VCHAR character.
func isVchar(r rune) bool {
	// Visible (printing) characters
	return '!' <= r && r <= '~' || isMultibyte(r)
}

// isMultibyte reports whether r is a multi-byte UTF-8 character
// as supported by RFC 6532
func isMultibyte(r rune) bool {
	return r >= utf8.RuneSelf
}

func (p *headerParser) parseNoFoldLiteral() (string, error) {
	if !p.consume('[') {
		return "", fmt.Errorf("missing '[' in no-fold-literal")
	}

	i := 0
	for {
		r, size := utf8.DecodeRuneInString(p.s[i:])
		if size == 1 && r == utf8.RuneError {
			return "", fmt.Errorf("invalid UTF-8 in no-fold-literal: %q", p.s)
		} else if size == 0 || !isDtext(r) {
			break
		}
		i += size
	}
	var lit string
	lit, p.s = p.s[:i], p.s[i:]

	if !p.consume(']') {
		return "", fmt.Errorf("missing ']' in no-fold-literal")
	}
	return "[" + lit + "]", nil
}

func isDtext(r rune) bool {
	switch r {
	case '[', ']', '\\':
		return false
	}
	return isVchar(r)
}

// isQtext reports whether r is an RFC 5322 qtext character.
func isQtext(r rune) bool {
	// Printable US-ASCII, excluding backslash or quote.
	if r == '\\' || r == '"' {
		return false
	}
	return isVchar(r)
}

// isWSP reports whether r is a WSP (white space).
// WSP is a space or horizontal tab (RFC 5234 Appendix B).
func isWSP(r rune) bool {
	return r == ' ' || r == '\t'
}

func (p *headerParser) consumeQuotedString() (qs string, err error) {
	// Assume first byte is '"'.
	i := 1
	qsb := make([]rune, 0, 10)

	escaped := false

Loop:
	for {
		r, size := utf8.DecodeRuneInString(p.s[i:])

		switch {
		case size == 0:
			return "", fmt.Errorf("unclosed quoted-string")

		case size == 1 && r == utf8.RuneError:
			return "", fmt.Errorf("invalid utf-8 in quoted-string: %q", p.s)

		case escaped:
			//  quoted-pair = ("\" (VCHAR / WSP))

			if !isVchar(r) && !isWSP(r) {
				return "", fmt.Errorf("bad character in quoted-string: %q", r)
			}

			qsb = append(qsb, r)
			escaped = false

		case isQtext(r) || isWSP(r):
			// qtext (printable US-ASCII excluding " and \), or
			// FWS (almost; we're ignoring CRLF)
			qsb = append(qsb, r)

		case r == '"':
			break Loop

		case r == '\\':
			escaped = true

		default:
			return "", fmt.Errorf("bad character in quoted-string: %q", r)

		}

		i += size
	}
	p.s = p.s[i+1:]
	return string(qsb), nil
}

/*
   As per RFC 5322 3.6.4
   msg-id/c-id          =   [CFWS] "<" id-left "@" id-right ">" [CFWS]
   where:

   id-left         =   dot-atom-text / obs-id-left
   id-right        =   dot-atom-text / no-fold-literal / obs-id-right
   no-fold-literal =   "[" *dtext "]"

   But it's common to see these
   without <>
   without @
   with () etc
   With quoted text
   parseMsgIDLin is more lineant in parsing msg-id
*/

func (p *headerParser) parseMsgIDLin(requireAngleAddr bool) (parsedMsgId string, err error) {

	angleAddr := false

	var left, right string

	if !p.skipCFWS() {
		return parsedMsgId, fmt.Errorf("malformed parenthetical comment")
	}

	if requireAngleAddr && p.consume('<') {
		return "", fmt.Errorf("missing '<' in msg-id")
	} else {
		if p.consume('<') {
			angleAddr = true
		}
	}

	// Parse the left side
	// Could be a quoted string or a dot-atom-text
	if !p.empty() {
		if p.peek() == '"' {
			left, err = p.consumeQuotedString()
			if left == "" {
				err = fmt.Errorf("empty quoted string in addr-spec")
			}
			if err != nil {
				err = fmt.Errorf("Error parsing quoted string: %v", err)
			}
		} else {
			// Else it's a dot-atom
			left, err = p.consumeAtomText(true, true, false)
			if err != nil {
				err = fmt.Errorf("Error parsing quoted string: %v", err)
			}
		}
	} else {
		err = fmt.Errorf("Error empty value")
	}

	if err != nil {
		return parsedMsgId, err
	}

	// Get the right

	if p.consume('@') {
		if !p.skipCFWS() {
			return parsedMsgId, fmt.Errorf("malformed parenthetical comment after @")
		}

		if !p.empty() && p.peek() == '[' {
			// no-fold-literal
			right, err = p.parseNoFoldLiteral()
		} else {
			right, err = p.consumeAtomText(true, true, true)
			if err != nil {
				// This is added for supporting ids like <local-part@domain1@domain2>
				// refer mimeKit's parser for more details
				if err.Error() == "Err Empty string" {
					err = nil
					right = ""
				} else {
					err = fmt.Errorf("malformed atom after @: %v", err)
				}
			}
		}
	}

	if err != nil {
		return parsedMsgId, err
	}

	if angleAddr {
		if !p.consume('>') {
			err = fmt.Errorf("missing '>' in msg-id")
		}
	}

	if right == "" {
		parsedMsgId = left
	} else {
		parsedMsgId = left + "@" + right
	}

	parsedMsgId = "<" + parsedMsgId + ">"

	return parsedMsgId, err

}

func (p *headerParser) parseMsgID() (string, error) {
	if !p.skipCFWS() {
		return "", fmt.Errorf("malformed parenthetical comment")
	}

	if !p.consume('<') {
		return "", fmt.Errorf("missing '<' in msg-id")
	}

	left, err := p.consumeAtomText(true, false, false)
	if err != nil {
		return "", err
	}

	if !p.consume('@') {
		return "", fmt.Errorf("missing '@' in msg-id")
	}

	var right string
	if !p.empty() && p.peek() == '[' {
		// no-fold-literal
		right, err = p.parseNoFoldLiteral()
	} else {
		right, err = p.consumeAtomText(true, false, false)
		if err != nil {
			return "", err
		}
	}

	if !p.consume('>') {
		return "", fmt.Errorf("missing '>' in msg-id")
	}

	if !p.skipCFWS() {
		return "", fmt.Errorf("malformed parenthetical comment")
	}

	return left + "@" + right, nil
}

// MsgIDList parses a list of message identifiers. It returns message
// identifiers without angle brackets. If the header field is missing, it
// returns nil.
//
// This can be used on In-Reply-To and References header fields.
func MsgIDList(v string) ([]string, error) {
	if v == "" {
		return nil, nil
	}

	p := headerParser{v}
	var l []string
	for !p.empty() {
		msgID, err := p.parseMsgIDLin(false)
		if err != nil {
			return l, err
		}

		// TODO: Check if this is needed
		//msgID = FromIDHeader(msgID)

		l = append(l, msgID)
	}

	return l, nil
}

// FromIDHeader decodes a Content-ID or Message-ID header value (RFC 2392) into a utf-8 string.
// Example: "<foo%3fbar+baz>" becomes "foo?bar baz".
func FromIDHeader(v string) string {
	if v == "" {
		return v
	}
	v = strings.TrimLeft(v, "<")
	v = strings.TrimRight(v, ">")
	if r, err := url.QueryUnescape(v); err == nil {
		v = r
	}
	return v
}

// ToIDHeader encodes a Content-ID or Message-ID header value (RFC 2392) from a utf-8 string.
func ToIDHeader(v string) string {
	v = url.QueryEscape(v)
	return "<" + strings.Replace(v, "%40", "@", -1) + ">"
}
