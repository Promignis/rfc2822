package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/Promignis/rfc2822"
)

func main() {
	reader := bytes.NewBuffer(encodedEml)

	callback := func(n *rfc2822.Node) error {
		buf, err := ioutil.ReadAll(n)
		if err != nil {
			return err
		}

		n.Body = append(n.Body, string(buf))

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
