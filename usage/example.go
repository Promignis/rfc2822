package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/Promignis/rfc2822"
)

func main() {
	reader := bytes.NewBuffer(encodedEml)

	mimeTree := rfc2822.NewMimeTree(reader)

	callback := func(r io.Reader, n *rfc2822.Node) error {
		buf, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}

		n.Body = append(n.Body, string(buf))

		return nil
	}

	err := mimeTree.Parse(callback)

	if err != nil {
		fmt.Println("error while parsing", err)
	}

	mimeTree.Finalize()

	root := mimeTree.MimetreeRoot.ChildNodes[0]

	jsonVal, err := json.Marshal(root)
	if err != nil {
		fmt.Println("error while marshaling", err)
	}

	fmt.Println(string(jsonVal))
}
