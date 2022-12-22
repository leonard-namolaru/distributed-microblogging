/*
$ curl https://jch.irif.fr:8443/udp-address.json -i -k
HTTP/2 200
cache-control: no-cache
content-type: application/json
content-length: 37
date: Tue, 08 Nov 2022 19:51:01 GMT

{"host":"81.194.27.155","port":8443}
*/

// The source of the comments: the official documentation of GO
package main

import (
	// Package json implements encoding and decoding of JSON
	"encoding/json"

	// Package crypto collects common cryptographic constants.
	// Package tls partially implements TLS 1.2, as specified in RFC 5246, and TLS 1.3, as specified in RFC 8446.
	"crypto/tls"

	// Package fmt implements formatted I/O with functions analogous to C's printf and scanf.
	"fmt"

	// Package io provides basic interfaces to I/O primitives.
	// Package ioutil implements some I/O utility functions.
	"io/ioutil"

	// Package log implements a simple logging package.
	"log"

	// Package net provides a portable interface for network I/O,
	// including TCP/IP, UDP, domain name resolution, and Unix domain sockets.
	// Package http provides HTTP client and server implementations.
	"net/http"

	// Package time provides functionality for measuring and displaying time.
	"time"
	// Package os provides a platform-independent interface to operating system functionality.
)

const URL = "https://127.0.0.1:8443/udp-address.json"

type Response struct {
	Host string `json:"host"`
	Port int64  `json:"port"`
}

func getHttpResponse(client *http.Client, requestUrl string) []byte {
	fmt.Printf("HTTP GET REQUEST : %v \n", requestUrl)

	// func http.NewRequest(method string, url string, body io.Reader) (*http.Request, error)
	req, errorMessage := http.NewRequest("GET", requestUrl, nil)
	if errorMessage != nil {
		// func log.Fatal(v ...any)
		// Fatal is equivalent to Print() followed by a call to os.Exit(1).
		log.Fatal("http.NewRequest() function : ", errorMessage)
	}

	// func (*http.Client).Do(req *http.Request) (*http.Response, error)
	r, errorMessage := client.Do(req)
	if errorMessage != nil {
		log.Fatal("client.Do() function : ", errorMessage)
	}

	// func ioutil.ReadAll(r io.Reader) ([]byte, error)
	body, errorMessage := ioutil.ReadAll(r.Body)
	// func (io.Closer).Close() error
	r.Body.Close()

	if errorMessage != nil {
		log.Fatal("ioutil.ReadAll() function : ", errorMessage)
	}

	return body
}

func main() {
	transport := &*http.DefaultTransport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // This is a code for pedagogical purposes !
	client := &http.Client{
		Transport: transport,
		Timeout:   50 * time.Second,
	}

	// func getHttpResponse(client *http.Client, requestUrl string) []byte
	body := getHttpResponse(client, URL)
	for _, char := range body {
		fmt.Printf("%v", string(char))
	}

	fmt.Printf("\n")

	var udpAddress Response

	// func json.Unmarshal(data []byte, v any) error
	// Unmarshal parses the JSON-encoded data and stores the result in the value pointed to by v
	errorMessage := json.Unmarshal(body, &udpAddress)
	if errorMessage != nil {
		log.Fatal("json.Unmarshal() function : ", errorMessage)
	}

	fmt.Printf("Host : %s \n", udpAddress.Host)
	fmt.Printf("Port : %d \n", udpAddress.Port)
}
