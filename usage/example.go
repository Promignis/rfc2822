package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/Promignis/rfc2822"
)

func main() {
	reader := bytes.NewBuffer(TestEml2)

	mimeTree := rfc2822.NewMimeTree(reader)

	err := mimeTree.Parse()

	if err != nil {
		fmt.Println("error while parsing", err)
	}

	mimeTree.Finalize()

	root := mimeTree.MimetreeRoot

	jsonVal, err := json.Marshal(root)
	if err != nil {
		fmt.Println("error while marshaling", err)
	}

	fmt.Println(string(jsonVal))
}
