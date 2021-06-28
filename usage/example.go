package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	mime "github.com/Promignis/rfc2822"
)

func main() {
	reader := bytes.NewBuffer(TestEml2NoEpilogue)

	sm := mime.NewFormattedRootHeaders()

	smCallback := bodyCallback()
	hc := mime.GetRootHeaderCallback(&sm)

	treeRoot, err := mime.ParseMime(reader, smCallback, hc, false)

	fmt.Println("========= SM ============")
	fmt.Println(sm.Date)
	fmt.Println(sm.Subject)
	fmt.Println("sm.From", sm.From)
	fmt.Println("to", sm.To)
	fmt.Println("bcc", sm.Bcc)
	fmt.Println("cc", sm.Cc)
	fmt.Println("contentType", sm.ContentType)
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

func bodyCallback() func(n *mime.Node) error {
	return func(n *mime.Node) error {
		// Is this node an attachment
		isAttachment := true

		if n.ContentType.Type == "text" {
			if n.ContentType.SubType == "plain" || n.ContentType.SubType == "html" {
				isAttachment = false
			}
		} else if n.ContentDisposition.MediaType != "inline" {
			isAttachment = false
		}

		// Content Type can be of the following types
		// message, multipart, text, application
		// for now anything aside from
		/*
			text/
				plain
				html
			multipart/
				mixed
				alternate
				related
		*/
		// is not processed , they will be treated as attachments/blob

		// If of type html/text
		// keep putting in body till a limit, if limit crossed then create a new reader and put in store
		// if text then keep a small subset?
		// update the atts array
		// if attachment
		// put in store
		// update the atts array

		buf, err := ioutil.ReadAll(n)
		if err != nil {
			return err
		}
		fmt.Println("\nbody..........................")
		fmt.Println("attachments: ", isAttachment)
		fmt.Println(string(buf))
		fmt.Println("body over..................\n")

		return nil
	}
}
