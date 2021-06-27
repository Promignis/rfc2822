package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"

	mime "github.com/Promignis/rfc2822"
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

	fmt.Println("========= SM ============")
	fmt.Println(sm.Date)
	fmt.Println(sm.Subject)
	fmt.Println("sm.From", sm.From)
	fmt.Println("to", sm.To)
	fmt.Println("bcc", sm.Bcc)
	fmt.Println("cc", sm.Cc)
	fmt.Println(sm.HTML)
	fmt.Println(sm.Text)
	fmt.Println("ref", sm.References)
	fmt.Println(sm.InReplyTo)
	fmt.Println("msgid:", sm.MessageID)

	if err != nil {
		log.Fatal("error while parsing", err)
	}

	jsonVal, err := json.Marshal(treeRoot)
	if err != nil {
		fmt.Println("error while marshaling", err)
	}

	fmt.Println(string(jsonVal))
}
