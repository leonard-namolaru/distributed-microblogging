/**
$ curl -X POST https://jch.irif.fr:8443/get-token -H 'Content-Type: application/json' -d '{ "username": "xxx", "password": "Rosebud" }' -i -k
HTTP/2 200
cache-control: no-cache
content-type: aplication/jwt
content-length: 147
date: Tue, 25 Oct 2022 18:29:02 GMT

eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdWQiOlsieHh4Il0sImV4cCI6MTY2NjcyMjU3MiwiaWF0IjoxNjY2NzIyNTQyfQ.LX7ZX0gb-Q6I8q2qVEu5D5Z5X9StdxMaYgKwdSMMHWY

----------------------------------------------------------------------------------------------------------------------------------------------------------

$ curl -X POST https://jch.irif.fr:8443/get-token -H 'Content-Type: application/json' -d '{ "username": "xxx", "password": "YYY" }' -i -k
HTTP/2 401
content-type: text/plain; charset=utf-8
x-content-type-options: nosniff
content-length: 13
date: Tue, 25 Oct 2022 18:29:37 GMT

bad password
**/

// The source of the comments: the official documentation of GO
package main

// An import declaration states that the source file containing the declaration depends on functionality of the imported package
import (
	// Package strings implements simple functions to manipulate UTF-8 encoded strings.
	"strings"

	// Package crypto collects common cryptographic constants.
	// Package tls partially implements TLS 1.2, as specified in RFC 5246, and TLS 1.3, as specified in RFC 8446.
	"crypto/tls"

	// Package fmt implements formatted I/O with functions analogous to C's printf and scanf.
	"fmt"

	// Package log implements a simple logging package.
	"log"

	// Package net provides a portable interface for network I/O,
	// including TCP/IP, UDP, domain name resolution, and Unix domain sockets.
	// Package http provides HTTP client and server implementations.
	"net/http"

	// Package time provides functionality for measuring and displaying time.
	"time"

	// Package io provides basic interfaces to I/O primitives.
	// Package ioutil implements some I/O utility functions.
	"io/ioutil"

	// Package json implements encoding and decoding of JSON as defined in RFC 7159
	"encoding/json"
)

const HOST_URL = "https://jch.irif.fr:8443"
const EXIT_FAILURE = 1
const EXIT_SUCCESS = 0

type jsonMessage struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func httpPostRequest(client *http.Client, requestUrl string, msg string) []byte {
	fmt.Printf("HTTP POST REQUEST : %v \n", requestUrl)

	// func http.NewRequest(method string, url string, body io.Reader) (*http.Request, error)
	req, errorMessage := http.NewRequest("POST", requestUrl, strings.NewReader(msg))

	if errorMessage != nil {
		// Fatal is equivalent to Print() followed by a call to os.Exit(1).
		log.Fatal("http.NewRequest() function : ", errorMessage)
	}

	// func (http.Header).Add(key string, value string)
	req.Header.Add("Content-Type", "application/json")

	// func (*http.Client).Do(req *http.Request) (*http.Response, error)
	response, errorMessage := client.Do(req)
	if errorMessage != nil {
		log.Fatal("client.Do() function : ", errorMessage)
	}

	// func ioutil.ReadAll(r io.Reader) ([]byte, error)
	responseBody, errorMessage := ioutil.ReadAll(response.Body)

	if errorMessage != nil {
		log.Fatal("ioutil.ReadAll() function : ", errorMessage)
	}

	// func (io.Closer).Close() error
	response.Body.Close()

	return responseBody
}

func main() {
	transport := &*http.DefaultTransport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // This is a code for pedagogical purposes !
	client := &http.Client{
		Transport: transport,
		Timeout:   50 * time.Second,
	}

	jsonOfNewMessage, _ := json.Marshal(jsonMessage{Username: "XXX", Password: "Rosebud"})
	stringOfJason := string(jsonOfNewMessage)

	// func httpPostRequest(client *http.Client, requestUrl string, msg string) []byte
	httpResponseBody := httpPostRequest(client, HOST_URL+"/get-token", stringOfJason)

	fmt.Printf("%s", httpResponseBody)
}
