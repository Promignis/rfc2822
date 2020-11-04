package rfc2822

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"regexp"
	"strings"
)

var whitespace, linebreak, notNormalHeaderKey, newlineOrcarriage *regexp.Regexp

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
	newlineOrcarriage = regexp.MustCompile(`\n?\r`)
	notNormalHeaderKey = regexp.MustCompile(`[^a-zA-Z0-9\-*]`)
}

const MAX_MIME_NODES = 99

const HEADER = "header"
const BODY = "body"

type node struct {
	ChildNodes []*node
	Header     []string
	// https://golang.org/pkg/net/textproto/#MIMEHeader
	ParsedHeader   map[string][]string
	Body           []string
	Multipart      string
	Boundary       string
	LineCount      int
	Size           int
	root           bool
	state          string
	parentNode     *node
	parentBoundary string
}

type MimeTree struct {
	rawScanner   *bufio.Scanner
	rawReader    *bufio.Reader
	MimetreeRoot *node
	nodeCount    int16
	currentNode  *node
}

func NewMimeTree(raw io.Reader) *MimeTree {

	rootNode := node{
		root:           true,
		ChildNodes:     []*node{},
		state:          "",
		Header:         []string{},
		ParsedHeader:   map[string][]string{},
		Body:           []string{},
		Multipart:      "",
		parentBoundary: "",
		Boundary:       "",
		parentNode:     nil,
		LineCount:      0,
		Size:           0,
	}

	mimeTree := MimeTree{
		rawScanner:   bufio.NewScanner(raw),
		rawReader:    bufio.NewReader(raw),
		MimetreeRoot: &rootNode,
		nodeCount:    0,
		currentNode:  nil,
	}

	mimeTree.currentNode = mimeTree.createNode(&rootNode)

	return &mimeTree
}

func (mt *MimeTree) createNode(parent *node) *node {
	mt.nodeCount++

	newNode := node{
		state:          HEADER,
		ChildNodes:     []*node{},
		Header:         []string{},
		ParsedHeader:   map[string][]string{},
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

type ParserCallback func(interface{}) error

func readNextLine(r *bufio.Reader, l []byte) ([]byte, error) {
	for {
		lineb, more, err := r.ReadLine()

		if err != nil {
			return l, err
		}

		l = append(l, lineb...)

		if !more {
			break
		}
	}

	return l, nil

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

		nextLine, err := readNextLine(mt.rawReader, nil)

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
			if line == "" {
				mt.processHeader()
				mt.processContentType()
				mt.currentNode.state = BODY
			} else {
				mt.currentNode.Header = append(mt.currentNode.Header, line)
			}

			break

		case BODY:

			if (mt.currentNode.parentBoundary != "") && (line == "--"+mt.currentNode.parentBoundary || line == "--"+mt.currentNode.parentBoundary+"--") {
				if line == "--"+mt.currentNode.parentBoundary {
					mt.currentNode = mt.createNode(mt.currentNode.parentNode)
				} else {
					mt.currentNode = mt.currentNode.parentNode
				}
			} else if (mt.currentNode.Boundary != "") && (line == "--"+mt.currentNode.Boundary) {
				mt.currentNode = mt.createNode(mt.currentNode)
			} else {
				if mt.currentNode.parentBoundary != "" {
					bodReader := newBodyReader(mt.currentNode.parentBoundary, mt.rawReader)
					buf, err := ioutil.ReadAll(bodReader)
					if err != nil {
						return err
					}
					buf = append([]byte(line), buf...)

					mt.currentNode.Body = append(mt.currentNode.Body, string(buf))

				} else if mt.currentNode.parentBoundary == "" {

					buf, err := ioutil.ReadAll(mt.rawReader)
					if err != nil {
						return err
					}
					buf = append([]byte(line), buf...)

					mt.currentNode.Body = append(mt.currentNode.Body, string(buf))
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

func (mt *MimeTree) processHeader() {
	var key, value string

	headers := mt.currentNode.Header

	for i := (len(headers) - 1); i >= 0; i-- {
		if i > 0 && whitespace.Match([]byte(headers[i])) {
			headers[i-1] = headers[i-1] + "\r\r" + headers[i]
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

			// Do not touch headers that have strange looking keys, keep these
			// only in the unparsed array
			if notNormalHeaderKey.Match([]byte(key)) || len(key) > 100 {
				continue
			}

			// TODO: Check if values are utf-7, what happens then
			value = string(linebreak.ReplaceAll([]byte(value), []byte(" ")))
			mt.currentNode.ParsedHeader[key] = append(mt.currentNode.ParsedHeader[key], value)
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

	/*
		TODO:
		Parse address, ie values for headers 'from', 'sender', 'reply-to', 'to', 'cc', 'bcc'
		they are of form 'Name <address@domain>'
		convert them to proper structure like for eg.
		 [{name: 'Name', address: 'address@domain'}]

	*/
}

func (mt *MimeTree) processContentType() error {

	if _, ok := mt.currentNode.ParsedHeader["content-type"]; !ok {
		return nil
	}

	// parse content type
	/*
		Certain headers like content-type has ; seperated params
		eg: Content-Type: multipart/mixed; boundary="000000000000ffd62a05b2a4b0bd"
		this function seperates out the params in a typed structure
	*/
	headerVal := mt.currentNode.ParsedHeader["content-type"][0]

	mediatype, params, err := mime.ParseMediaType(headerVal)

	if err != nil {
		return fmt.Errorf("Could not parse content-type header %v: %v", headerVal, err)
	}

	mediaTypeSplit := strings.Split(mediatype, "/")
	subtype := ""
	if len(mediaTypeSplit) == 2 {
		subtype = mediaTypeSplit[1]
	}

	if mediaTypeSplit[0] == "multipart" {
		if _, ok := params["boundary"]; ok {
			mt.currentNode.Multipart = subtype
			mt.currentNode.Boundary = params["boundary"]
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

	var walker func(n *node)

	walker = func(n *node) {
		// TODO: Handle content type 'message/rfc822'
		lc := 0
		size := 0
		if len(n.Body) != 0 {
			lc = len(n.Body) - 1

			for i, _ := range n.Body {
				// ensure proper line endings
				n.Body[i] = string(newlineOrcarriage.ReplaceAll([]byte(n.Body[i]), []byte("\n\r")))
				// Add the size
				size += len(n.Body[i])
			}
		}
		n.LineCount = lc
		n.Size = size

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
