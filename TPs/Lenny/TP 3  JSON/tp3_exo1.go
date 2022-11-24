// 3. Combien de RTT votre programme attend-il les données ? Comparez avec la version REST du TP précédent.
// This TP : 1 RTT, TP 2 : 2 RTT

// 4. Imaginez que le serveur soit modifié pour que les messages contiennent un champ supplémentaire,
// par exemple un champ nommé from qui contient le nom de l’utilisateur qui a posté le message.
// Votre programme continuera-t-il à fonctionner ?
// Oui, car json ignore les champs qu'il ne connait pas. Nn nouveau serveur peut donc marcher avec un client ancien. L'inverse ? Si le client est bien écrit

/* ***********************************************************************************

//  -k, --insecure      Allow insecure server connections when using SSL // This is a code for pedagogical purposes !
//  -i, --include       Include protocol response headers in the output
//  -d, --data <data>   HTTP POST data
// Source : $ curl --help

/* ***********************************************************************************
lenny@DESKTOP-DMJ749K:/mnt/c/Users/lenny$ curl https://localhost:8443/chat/messages.json -i
curl: (60) SSL certificate problem: unable to get local issuer certificate...

*************************************************************************************

lenny@DESKTOP-DMJ749K:/mnt/c/Users/lenny$ curl https://localhost:8443/chat/messages.json -i -k
HTTP/2 200
content-type: application/json
content-length: 3
date: Thu, 06 Oct 2022 09:08:23 GMT

[]

*************************************************************************************

lenny@DESKTOP-DMJ749K:/mnt/c/Users/lenny$ curl https://localhost:8443/chat/ -i -k -d "first message in the chat"
HTTP/2 204
location: /chat/2b2e1e1673e786ed4a6741ce55ec3d0b
date: Thu, 06 Oct 2022 09:08:32 GMT

*************************************************************************************

lenny@DESKTOP-DMJ749K:/mnt/c/Users/lenny$ curl https://localhost:8443/chat/messages.json -i -k
HTTP/2 200
content-type: application/json
etag: "57228ce95d9f73876e4151ffaee3cee4"
last-modified: Thu, 06 Oct 2022 11:08:32 GMT
content-length: 100
date: Thu, 06 Oct 2022 09:08:47 GMT

[{"id":"2b2e1e1673e786ed4a6741ce55ec3d0b","time":1665047312578,"body":"first message in the chat"}]
lenny@DESKTOP-DMJ749K:/mnt/c/Users/lenny$ curl https://localhost:8443/chat/ -i -k -d "Another message in the chat"

*************************************************************************************
HTTP/2 204
location: /chat/31cb677673e913d0f7f7b00d70502143
date: Thu, 06 Oct 2022 09:09:19 GMT

*************************************************************************************
lenny@DESKTOP-DMJ749K:/mnt/c/Users/lenny$ curl https://localhost:8443/chat/messages.json -i -k
HTTP/2 200
content-type: application/json
etag: "b80eaf3e8c1d3ab5fa1bb73ee27a917d"
last-modified: Thu, 06 Oct 2022 11:09:19 GMT
content-length: 200
date: Thu, 06 Oct 2022 09:09:23 GMT

[{"id":"2b2e1e1673e786ed4a6741ce55ec3d0b","time":1665047312578,"body":"first message in the chat"},{"id":"31cb677673e913d0f7f7b00d70502143","time":1665047359673,"body":"Another message in the chat"}]
lenny@DESKTOP-DMJ749K:/mnt/c/Users/lenny$

************************************************************************************* */

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
	"os"
)

const CHAT_URL = "https://localhost:8443/chat/messages.json"
const EXIT_FAILURE = 1

type Response struct {
	Id   string `json:"id"`
	Time int64  `json:"time"`
	Body string `json:"body"`
}

func getHttpResponse(client *http.Client, requestUrl string) []byte {
	fmt.Printf("HTTP GET REQUEST : %v \n", requestUrl)

	// func http.NewRequest(method string, url string, body io.Reader) (*http.Request, error)
	req, errorMessage := http.NewRequest("GET", requestUrl, nil)
	if errorMessage != nil {
		log.Printf("http.NewRequest() function : %v", errorMessage)
		// func os.Exit(code int)
		os.Exit(EXIT_FAILURE)
	}

	// func (*http.Client).Do(req *http.Request) (*http.Response, error)
	r, errorMessage := client.Do(req)
	if errorMessage != nil {
		log.Printf("client.Do() function : %v", errorMessage)
		// func os.Exit(code int)
		os.Exit(EXIT_FAILURE)
	}

	// func ioutil.ReadAll(r io.Reader) ([]byte, error)
	body, errorMessage := ioutil.ReadAll(r.Body)
	// func (io.Closer).Close() error
	r.Body.Close()

	if errorMessage != nil {
		// func log.Printf(format string, v ...any)
		log.Printf("ioutil.ReadAll() function : %v", errorMessage)
		// func os.Exit(code int)
		os.Exit(EXIT_FAILURE)
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
	body := getHttpResponse(client, CHAT_URL)
	for _, char := range body {
		fmt.Printf("%v", string(char))
	}

	fmt.Printf("\n")

	var messages []Response

	// func json.Unmarshal(data []byte, v any) error
	// Unmarshal parses the JSON-encoded data and stores the result in the value pointed to by v
	errorMessage := json.Unmarshal(body, &messages)
	if errorMessage != nil {
		// func log.Printf(format string, v ...any)
		log.Printf("json.Unmarshal() function : %v \n", errorMessage)
		// func os.Exit(code int)
		os.Exit(EXIT_FAILURE)
	}

	fmt.Printf("Messeges in the chat : %d \n", len(messages))

	for i := 0; i < len(messages); i++ {
		// func fmt.Printf(format string, a ...any) (n int, err error)
		fmt.Printf("Id %v: %v\n", i, messages[i].Body)
	}
}
