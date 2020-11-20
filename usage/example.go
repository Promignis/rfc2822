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

	callback := func(r io.Reader, n *rfc2822.Node) error {
		buf, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}

		n.Body = append(n.Body, string(buf))

		fmt.Println(n.Body)

		return nil
	}

	treeRoot, err := rfc2822.ParseMime(reader, callback)

	if err != nil {
		fmt.Println("error while parsing", err)
	}

	jsonVal, err := json.Marshal(treeRoot)
	if err != nil {
		fmt.Println("error while marshaling", err)
	}

	fmt.Println(string(jsonVal))
}
