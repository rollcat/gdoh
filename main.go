package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
)

// DoHClient implements a DNS over HTTPS client.
//
// It supports querying via the "DNS wire" format via RawQuery (in
// which case, you are responsible for supplying the correct payload
// and interpreting the response), or the "DNS-JSON" format via Query.
type DoHClient struct {
	*http.Client
	Endpoints []string
}

// ErrResolver signifies an internal resolver error.
var ErrResolver = errors.New("Resolver error")

// The DNS-JSON format uses the "human readable" record type names in
// requests, but the wire format's opaque numbers in responses; we
// need to translate them back.
//
// https://www.iana.org/assignments/dns-parameters/dns-parameters.xhtml
var typeNameToNumber = map[string]int{
	"A":     1,
	"NS":    2,
	"CNAME": 5,
	"SOA":   6,
	"PTR":   12,
	"MX":    15,
	"TXT":   16,
	"AAAA":  28,
	"SRV":   33,
}

// pickEndpoint chooses an endpoint at random, so that 1. we
// load-balance; 2. we do not send 100% of our DNS traffic to a single
// entity.
func (c *DoHClient) pickEndpoint() string {
	return c.Endpoints[rand.Int()%len(c.Endpoints)]
}

// RawQuery performs a raw DNS query, using the wire format.
func (c *DoHClient) RawQuery(query []byte) ([]byte, error) {
	r, err := c.Client.Post(
		c.pickEndpoint(),
		"application/dns-udpwireformat",
		bytes.NewBuffer(query),
	)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		log.Printf("response: %#v", r)
		return nil, ErrResolver
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// Query performs a DNS-JSON query.
func (c *DoHClient) Query(name, type_ string) ([]string, error) {
	if _, ok := typeNameToNumber[type_]; !ok {
		return nil, ErrResolver
	}
	u, err := url.Parse(c.pickEndpoint())
	if err != nil {
		panic(err)
	}
	u.RawQuery = fmt.Sprintf(
		"name=%s&type=%s",
		url.QueryEscape(name),
		url.QueryEscape(type_),
	)
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/dns-json")
	r, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		log.Printf("response: %#v", r)
		return nil, ErrResolver
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	var v struct {
		Answer []struct {
			Type int
			Data string
		}
	}
	json.Unmarshal(body, &v)
	answers := []string{}
	for _, a := range v.Answer {
		if a.Type == typeNameToNumber[type_] {
			answers = append(answers, a.Data)
		}
	}
	return answers, nil
}

var dohClient = &DoHClient{
	Client: &http.Client{},
	Endpoints: []string{
		"https://1.0.0.1/dns-query",
		"https://1.1.1.1/dns-query",
	},
}

var listen = flag.String("listen", ":53", "UDP address to listen on")

func main() {
	if len(dohClient.Endpoints) == 0 {
		log.Fatal("No endpoints configured")
	}
	flag.Parse()
	laddr, err := net.ResolveUDPAddr("udp", *listen)
	if err != nil {
		log.Fatal(err)
	}
	ln, err := net.ListenUDP("udp", laddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Listening on %s", laddr.String())
	defer ln.Close()

	for {
		query := make([]byte, 128)
		n, _, _, addr, err := ln.ReadMsgUDP(query, nil)
		if err != nil {
			log.Print("read error:", err.Error())
			continue
		}
		query = query[:n]

		go func(query []byte, addr *net.UDPAddr) {
			var err error
			resp, err := dohClient.RawQuery(query)
			if err != nil {
				log.Print("query error:", err.Error())
				// TODO: how to tell client we've got an error?
				resp = []byte{0}
			}
			_, _, err = ln.WriteMsgUDP(resp, nil, addr)
			if err != nil {
				log.Print("write error:", err.Error())
			}
		}(query, addr)
	}
}
