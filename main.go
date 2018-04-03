package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/http"
)

// DoHClient implements a DNS over HTTPS client.
type DoHClient struct {
	http.Client
}

// ErrResolver signifies an internal resolver error.
var ErrResolver = errors.New("Resolver error")

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
	if r.StatusCode != 200 {
		log.Printf("response: %#v", r)
		return nil, ErrResolver
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
	ln, err := net.ListenUDP("udp", &net.UDPAddr{Port: 1253})
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	query := make([]byte, 128)
	n, _, _, addr, err := ln.ReadMsgUDP(query, nil)
	if err != nil {
		panic(err)
	}
	query = query[:n]

	resp, err := dohClient.RawQuery(query)
	if err != nil {
		panic(err)
	}
	_, _, err = ln.WriteMsgUDP(resp, nil, addr)
	if err != nil {
		panic(err)
	}
}
