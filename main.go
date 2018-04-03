package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
)

// DoHClient implements a DNS over HTTPS client.
type DoHClient struct {
	http.Client
}

// RawQuery performs a raw DNS query, using the wire format.
func (c *DoHClient) RawQuery(query []byte) ([]byte, error) {
	r, err := c.Client.Post(
		"https://1.1.1.1/dns-query",
		"application/dns-udpwireformat",
		bytes.NewBuffer(query),
	)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

var dohClient = &DoHClient{}

func main() {
	query := []byte{
		0x00, 0x00, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x03, 0x77, 0x77, 0x77,
		0x07, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65,
		0x03, 0x63, 0x6f, 0x6d, 0x00, 0x00, 0x01, 0x00,
		0x01,
	}
	log.Printf("query: %#v", query)
	body, err := dohClient.RawQuery(query)
	if err != nil {
		panic(err)
	}
	log.Printf("body: %#v", body)
}
