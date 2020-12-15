package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Promignis/mime"
)

func main() {
	reader := bytes.NewBuffer(encodedEml)

	// callback := func(n *mime.Node) error {
	// 	buf, err := ioutil.ReadAll(n)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	n.Body = append(n.Body, string(buf))

	// 	fmt.Println(n.Body, "------------")

	// 	return nil
	// }

	sm := mime.NewStructuredMime()

	dummyStore := newSampleStore()

	smCallback := mime.GetStorageCallback(&sm, dummyStore)
	hc := mime.GetRootHeaderCallback(&sm)

	treeRoot, err := mime.ParseMime(reader, smCallback, hc)

	if err != nil {
		log.Fatal("error while parsing", err)
	}

	jsonVal, err := json.Marshal(treeRoot)
	if err != nil {
		fmt.Println("error while marshaling", err)
	}

	fmt.Println(string(jsonVal))
}
