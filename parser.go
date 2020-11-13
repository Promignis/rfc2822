package rfc2822

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var whitespace, linebreak, notNormalHeaderKey *regexp.Regexp

var singleValueFields = []string{
	"content-tansfer-encoding",
	"content-id",
	"content-description",
	"content-language",
	"content-md5",
	"content-location",
	"content-type",
	"content-disposition",
}

func init() {
	// Compile all the regex
	whitespace = regexp.MustCompile(`^\s`)
	linebreak = regexp.MustCompile(`\s*\r?\n\s*`)
	notNormalHeaderKey = regexp.MustCompile(`[^a-zA-Z0-9\-*]`)
}

const MAX_MIME_NODES = 99
const MAX_HEADER_LINES = 1000
const MAX_LINE_OCTETS = 4000

const HEADER = "header"
const BODY = "body"

type Node struct {
	ChildNodes []*Node
	Headers    []string
	// https://golang.org/pkg/net/textproto/#MIMEHeader
	ParsedHeader       map[string][]string
	BadHeaders         map[string][]string
	Body               []string
	Multipart          string
	ContentType        ContentType
	ContentDisposition ContentDisposition
	Boundary           string
	LineCount          int
	Size               int
	root               bool
	state              string
	parentNode         *Node
	parentBoundary     string
}

type MimeTree struct {
	rawReader    *bufio.Reader
	MimetreeRoot *Node
	nodeCount    int16
	currentNode  *Node
}

type ContentType struct {
	Type    string
	SubType string
	Params  map[string]string
}

type ContentDisposition struct {
	MediaType string
	Params    map[string]string
}

func NewMimeTree(raw io.Reader) *MimeTree {

	rootNode := Node{
		root:           true,
		ChildNodes:     []*Node{},
		state:          "",
		Headers:        []string{},
		ParsedHeader:   map[string][]string{},
		BadHeaders:     map[string][]string{},
		Body:           []string{},
		Multipart:      "",
		parentBoundary: "",
		Boundary:       "",
		parentNode:     nil,
		LineCount:      0,
		Size:           0,
	}

	mimeTree := MimeTree{
		rawReader:    bufio.NewReader(raw),
		MimetreeRoot: &rootNode,
		nodeCount:    0,
		currentNode:  nil,
	}

	mimeTree.currentNode = mimeTree.createNode(&rootNode)

	return &mimeTree
}

func (mt *MimeTree) createNode(parent *Node) *Node {
	mt.nodeCount++

	newNode := Node{
		state:          HEADER,
		ChildNodes:     []*Node{},
		Headers:        []string{},
		ParsedHeader:   map[string][]string{},
		BadHeaders:     map[string][]string{},
		Body:           []string{},
		Multipart:      "",
		parentBoundary: parent.Boundary,
		Boundary:       "",
		parentNode:     parent,
		LineCount:      0,
		Size:           0,
	}

	parent.ChildNodes = append(parent.ChildNodes, &newNode)

	return &newNode
}

type ParserCallback func(bodyReader io.Reader, mimeNode *Node) error

func readNextLine(r *bufio.Reader, l []byte) ([]byte, []byte, error) {

	br := []byte("\n")

	l, err := readBytesWithLimit(r, byte('\n'), MAX_LINE_OCTETS)

	if err != nil {
		return l, br, err
	}

	lLen := len(l)

	if lLen >= 2 && l[lLen-2] == byte('\r') {
		br = []byte("\r\n")
	}

	return l, br, nil

}

type BodyReader struct {
	pboundary string
	bufReader *bufio.Reader
	n         int
	err       error
	readErr   error
}

func newBodyReader(boundary string, r io.Reader) *BodyReader {
	return &BodyReader{
		pboundary: boundary,
		bufReader: bufio.NewReaderSize(r, 256),
	}
}

func (bodR *BodyReader) Read(d []byte) (int, error) {
	boundry := bodR.pboundary
	dashB := []byte("--" + boundry)
	br := bodR.bufReader

	// Read into buffer until we identify some data to return,
	// or we find a reason to stop (boundary or read error).
	for bodR.n == 0 && bodR.err == nil {
		peek, _ := br.Peek(br.Buffered())

		bodR.n, bodR.err = scanUntilBoundary(peek, dashB, bodR.readErr)

		if bodR.n == 0 && bodR.err == nil {
			_, bodR.readErr = br.Peek(len(peek) + 1)

			if bodR.readErr == io.EOF {
				bodR.readErr = io.ErrUnexpectedEOF
			}
		}
	}

	if bodR.n == 0 {
		return 0, bodR.err
	}
	n := len(d)

	if n > bodR.n {
		n = bodR.n
	}
	n, _ = br.Read(d[:n])
	bodR.n -= n
	if bodR.n == 0 {
		return n, bodR.err
	}

	return n, nil
}

func scanUntilBoundary(buf, dashBoundary []byte, readErr error) (int, error) {
	// Search for "--boundary".
	if i := bytes.Index(buf, dashBoundary); i >= 0 {

		switch matchAfterPrefix(buf[i:], dashBoundary, readErr) {
		case -1:
			return i + len(dashBoundary), nil
		case 0:
			return i, nil
		case +1:
			return i, io.EOF
		}
	}
	if bytes.HasPrefix(dashBoundary, buf) {
		return 0, readErr
	}

	i := bytes.LastIndexByte(buf, dashBoundary[0])
	if i >= 0 && bytes.HasPrefix(dashBoundary, buf[i:]) {
		return i, nil
	}
	return len(buf), readErr
}

func matchAfterPrefix(buf, prefix []byte, readErr error) int {
	if len(buf) == len(prefix) {
		if readErr != nil {
			return +1
		}
		return 0
	}
	c := buf[len(prefix)]
	if c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '-' {
		return +1
	}
	return -1
}

func (mt *MimeTree) Parse(pc ParserCallback) error {
	line := ""
	var readerr error = nil
	for readerr != io.EOF {

		nextLine, lineBreak, err := readNextLine(mt.rawReader, nil)

		readerr = err
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		line = string(nextLine)

		switch mt.currentNode.state {
		case HEADER:
			// This means end of a header section
			// and start of body
			if line == string(lineBreak) {
				err := mt.processHeader()
				if err != nil {
					return err
				}
				err = mt.processContentType()
				if err != nil {
					return err
				}

				mt.currentNode.state = BODY
			} else {
				mt.currentNode.Headers = append(mt.currentNode.Headers, line)
			}

			break

		case BODY:

			if (mt.currentNode.parentBoundary != "") && (line == "--"+mt.currentNode.parentBoundary+string(lineBreak) || line == "--"+mt.currentNode.parentBoundary+"--"+string(lineBreak)) {
				if line == "--"+mt.currentNode.parentBoundary+string(lineBreak) {
					mt.currentNode = mt.createNode(mt.currentNode.parentNode)
				} else {
					mt.currentNode = mt.currentNode.parentNode
				}
			} else if (mt.currentNode.Boundary != "") && (line == "--"+mt.currentNode.Boundary+string(lineBreak)) {
				mt.currentNode = mt.createNode(mt.currentNode)
			} else {
				if mt.currentNode.parentBoundary != "" {

					bodReader := newBodyReader(mt.currentNode.parentBoundary, mt.rawReader)

					fullReader := io.MultiReader(bytes.NewReader(nextLine), bodReader)

					err := pc(fullReader, mt.currentNode)

					if err != nil {
						return err
					}
				} else if mt.currentNode.parentBoundary == "" {

					fullReader := io.MultiReader(bytes.NewReader(nextLine), mt.rawReader)

					err := pc(fullReader, mt.currentNode)

					if err != nil {
						return err
					}
				}
			}

			break

		default:
			return fmt.Errorf("Unexpected state")

		}

		if mt.nodeCount > MAX_MIME_NODES {
			return fmt.Errorf("MAX_MIME_NODES count crossed")
		}

	}

	return nil
}

func (mt *MimeTree) processHeader() error {
	var key, value string

	headers := mt.currentNode.Headers

	for i := (len(headers) - 1); i >= 0; i-- {
		if i > 0 && whitespace.Match([]byte(headers[i])) {
			headers[i-1] = headers[i-1] + "\r\n" + headers[i]
			headers = headers[:i]
		} else {
			spl := strings.Split(headers[i], ":")
			if len(spl) >= 2 {
				key = strings.ToLower(strings.TrimSpace(spl[0]))
				value = strings.Join(spl[1:], ":")
			} else if len(spl) == 1 {
				key = strings.ToLower(strings.TrimSpace(spl[0]))
				value = ""
			}

			// TODO: Check if values are utf-7
			value = string(linebreak.ReplaceAll([]byte(value), []byte("")))

			// Track headers that have strange looking keys, keep these
			// in the seperate section
			if notNormalHeaderKey.Match([]byte(key)) || len(key) > 100 {
				mt.currentNode.BadHeaders[key] = append(mt.currentNode.ParsedHeader[key], value)
			} else {
				mt.currentNode.ParsedHeader[key] = append(mt.currentNode.ParsedHeader[key], value)
			}
		}
	}

	// Make sure Content-Type is always there
	if _, ok := mt.currentNode.ParsedHeader["content-type"]; !ok {
		mt.currentNode.ParsedHeader["content-type"] = []string{"text/plain"}
	}

	// Make sure following fields have only songle values
	for _, k := range singleValueFields {
		if _, ok := mt.currentNode.ParsedHeader[k]; ok {
			// Basically pop the last value
			mt.currentNode.ParsedHeader[k] = mt.currentNode.ParsedHeader[k][len(mt.currentNode.ParsedHeader[k])-1:]
		}
	}

	if contDisp, ok := mt.currentNode.ParsedHeader["content-disposition"]; ok {
		parsedContentDisp, err := ParseContentDisposition(contDisp[0])
		if err != nil {
			return fmt.Errorf("Could not parse content disposition %v: %v", contDisp[0], err)
		}
		mt.currentNode.ContentDisposition = parsedContentDisp
	}

	/*
		TODO:
		Parse address, ie values for headers 'from', 'sender', 'reply-to', 'to', 'cc', 'bcc'
		they are of form 'Name <address@domain>'
		convert them to proper structure like for eg.
		 [{name: 'Name', address: 'address@domain'}]

	*/

	return nil
}

func (mt *MimeTree) processContentType() error {

	if _, ok := mt.currentNode.ParsedHeader["content-type"]; !ok {
		return nil
	}

	parsedContentType, err := ParseContentType(mt.currentNode.ParsedHeader["content-type"][0])
	if err != nil {
		return fmt.Errorf("Could not parse content type %v: %v", mt.currentNode.ParsedHeader["content-type"][0], err)
	}
	mt.currentNode.ContentType = parsedContentType

	/*
		Certain headers like content-type has ; seperated params
		eg: Content-Type: multipart/mixed; boundary="000000000000ffd62a05b2a4b0bd"
		this function seperates out the params in a typed structure
	*/

	if mt.currentNode.ContentType.Type == "multipart" {
		if _, ok := mt.currentNode.ContentType.Params["boundary"]; ok {
			mt.currentNode.Multipart = mt.currentNode.ContentType.SubType
			mt.currentNode.Boundary = mt.currentNode.ContentType.Params["boundary"]
		}
	}
	return nil

}

// convert content-type: 'text/plain; charset=utf-8' -> {value: 'text/plain', params:{charset: 'utf-8'}}
func (mt *MimeTree) processHeaderValue() {

}

func (mt *MimeTree) Finalize() {
	if mt.currentNode.state == HEADER {
		mt.processHeader()
		mt.processContentType()
	}

	var walker func(n *Node)

	walker = func(n *Node) {
		// TODO: Handle content type 'message/rfc822'

		for _, cn := range n.ChildNodes {
			walker(cn)
		}

		// Empty out some unnecesary states
		n.parentNode = nil
		n.state = ""
		if len(n.ChildNodes) == 0 {
			n.ChildNodes = nil
		}
		n.parentBoundary = ""
	}

	walker(mt.MimetreeRoot)

	mt.currentNode = nil
}
