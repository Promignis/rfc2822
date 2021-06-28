package rfc2822

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var whitespace, linebreak *regexp.Regexp

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
}

const MAX_MIME_NODES = 99
const MAX_HEADER_LINES = 1000
const MAX_LINE_OCTETS = 4000

const HEADER = "header"
const BODY = "body"

var ValidContentDispositions = []string{"inline", "attachment"}

// For images with large attachemts this could cause heap churn. Take a look
const BufferReaderSize = 50 * 1024

type tempState struct {
	headerLines    []string
	root           bool
	state          string
	parentNode     *Node
	parentBoundary string
	bodyReader     io.Reader
}

type Node struct {
	ChildNodes []*Node
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
	tstate             tempState
}

func (n *Node) Read(d []byte) (int, error) {
	i, err := n.tstate.bodyReader.Read(d)
	n.Size += i
	// TODO: Put the bytes into node.Body for a particular config
	// right now callback has to explicitly put the body inside node.Body
	return i, err
}

type mimeTree struct {
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

func newMimeTree(raw io.Reader) *mimeTree {

	intialState := tempState{
		root:           true,
		state:          "",
		parentBoundary: "",
		parentNode:     nil,
		headerLines:    []string{},
	}

	rootNode := Node{
		ChildNodes:   []*Node{},
		ParsedHeader: map[string][]string{},
		BadHeaders:   map[string][]string{},
		Body:         []string{},
		Multipart:    "",
		Boundary:     "",
		LineCount:    0,
		Size:         0,
		tstate:       intialState,
	}

	mimeTree := mimeTree{
		rawReader:    bufio.NewReaderSize(raw, BufferReaderSize),
		MimetreeRoot: &rootNode,
		nodeCount:    0,
		currentNode:  nil,
	}

	mimeTree.currentNode = mimeTree.createNode(&rootNode)

	return &mimeTree
}

func (mt *mimeTree) createNode(parent *Node) *Node {
	mt.nodeCount++

	newTempState := tempState{state: HEADER,
		headerLines:    []string{},
		parentBoundary: parent.Boundary,
		parentNode:     parent}

	newNode := Node{
		ChildNodes:   []*Node{},
		BadHeaders:   map[string][]string{},
		Body:         []string{},
		Multipart:    "",
		ParsedHeader: map[string][]string{},
		Boundary:     "",
		LineCount:    0,
		Size:         0,
		tstate:       newTempState,
	}

	parent.ChildNodes = append(parent.ChildNodes, &newNode)

	return &newNode
}

type BodyCallback func(mimeNode *Node) error
type RootHeaderCallback func(node *Node) error

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

func (mt *mimeTree) parse(pc BodyCallback, hc RootHeaderCallback) error {
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

		switch mt.currentNode.tstate.state {
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

				// Call root header callback
				if mt.currentNode.tstate.parentNode.tstate.root && hc != nil {
					err := hc(mt.currentNode)
					if err != nil {
						return fmt.Errorf("Error parsing header: %v", err)
					}
				}

				mt.currentNode.tstate.state = BODY
			} else {
				mt.currentNode.tstate.headerLines = append(mt.currentNode.tstate.headerLines, line)

				if len(mt.currentNode.tstate.headerLines) > MAX_HEADER_LINES {
					return errMaxHeaderLines
				}
			}

			break

		case BODY:
			var fullReader io.Reader
			if (mt.currentNode.tstate.parentBoundary != "") && (line == "--"+mt.currentNode.tstate.parentBoundary+string(lineBreak) || line == "--"+mt.currentNode.tstate.parentBoundary+"--"+string(lineBreak)) {
				if line == "--"+mt.currentNode.tstate.parentBoundary+string(lineBreak) {
					mt.currentNode = mt.createNode(mt.currentNode.tstate.parentNode)
				} else {
					mt.currentNode = mt.currentNode.tstate.parentNode
				}
			} else if (mt.currentNode.Boundary != "") && (line == "--"+mt.currentNode.Boundary+string(lineBreak)) {
				mt.currentNode = mt.createNode(mt.currentNode)
			} else {
				if mt.currentNode.tstate.parentBoundary != "" {
					bodReader := newBodyReader(mt.currentNode.tstate.parentBoundary, mt.rawReader)
					fullReader = io.MultiReader(bytes.NewReader(nextLine), bodReader)
				} else if mt.currentNode.tstate.parentBoundary == "" {
					fullReader = io.MultiReader(bytes.NewReader(nextLine), mt.rawReader)
				}

				// Check for content transfer encoding
				if enc, ok := mt.currentNode.ParsedHeader["content-transfer-encoding"]; ok {
					if decodedReader, encErr := encodingReader(enc[0], fullReader); encErr != nil {
						return encErr
					} else {
						fullReader = decodedReader
					}
				}

				// TODO: Charset readers

				mt.currentNode.tstate.bodyReader = fullReader

				err := pc(mt.currentNode)

				if err != nil {
					return err
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

func (mt *mimeTree) processHeader() error {
	var key, value string

	headers := mt.currentNode.tstate.headerLines

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
				// len 1 means no ":" was found. This is a malformed header line
				// TODO: Maybe return a malformed header line error here
				key = strings.ToLower(strings.TrimSpace(spl[0]))
				value = ""
			}

			// TODO: Check if values are utf-7
			value = string(linebreak.ReplaceAll([]byte(value), []byte("")))
			value = strings.Trim(value, " ")

			// Track headers that have strange looking keys, keep these
			// in the seperate section
			validHeader := true
			for _, c := range []byte(key) {
				if !validHeaderKeyByte(c) {
					validHeader = false
					break
				}
			}

			if !validHeader || len(key) > 100 || key == "" {
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

	// Make sure following fields have only single values
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

		if !Contains(parsedContentDisp.MediaType, ValidContentDispositions) {
			return fmt.Errorf("Invalid content disposition value: %v", parsedContentDisp.MediaType)
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

func (mt *mimeTree) processContentType() error {

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
		} else {
			// No boundary found. Return error
			return errNoBoundary
		}
	}
	return nil

}

func (mt *mimeTree) finalize() {

	if mt.currentNode.tstate.state == HEADER {
		mt.processHeader()
		mt.processContentType()
	}

	var walker func(n *Node)

	walker = func(n *Node) {
		// TODO: Handle content type 'message/rfc822'

		for _, cn := range n.ChildNodes {
			walker(cn)
		}

		// Empty out temp states
		n.tstate.parentNode = nil
		n.tstate.state = ""
		n.tstate.headerLines = []string{}
		if len(n.ChildNodes) == 0 {
			n.ChildNodes = nil
		}
		n.tstate.parentBoundary = ""
		n.tstate.bodyReader = nil
	}

	walker(mt.MimetreeRoot)

	mt.currentNode = nil
}

func ParseMime(r io.Reader, bc BodyCallback, hc RootHeaderCallback) (*Node, error) {
	mimeTree := newMimeTree(r)

	err := mimeTree.parse(bc, hc)

	if err != nil {
		return &Node{}, err
	}

	mimeTree.finalize()

	root := mimeTree.MimetreeRoot.ChildNodes[0]

	return root, nil
}
