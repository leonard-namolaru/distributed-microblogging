// The source of the comments: the official documentation of GO

package main

import (
	// Package json implements encoding and decoding of JSON
	"encoding/json"
	"strconv"

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
const MESSEGES_COUNT = 4

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

	// func strconv.Itoa(i int) string
	// Itoa is equivalent to FormatInt(int64(i), 10).
	requestUrl := CHAT_URL + "?count=" + strconv.Itoa(MESSEGES_COUNT)

	// func getHttpResponse(client *http.Client, requestUrl string) []byte
	body := getHttpResponse(client, requestUrl)
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

	fmt.Printf("%d last messages : \n\n", MESSEGES_COUNT)

	for i := 0; i < len(messages); i++ {
		// func fmt.Printf(format string, a ...any) (n int, err error)
		fmt.Printf("Id %v: %v\n", i, messages[i].Body)
	}
}
