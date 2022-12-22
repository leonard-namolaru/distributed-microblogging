// The source of the comments: the official documentation of GO
// The problem with this solution - if a message is deleted from the chat,
// the program does not take this into account. A better solution is to store all the Ids of the chat messages in a array

// "Comparez avec l’approche REST" : In TP3 we use json coding (unlike TP2),
// therefore this is not a "pure" RESET, also, transferring the count parameter to the server is not a pure RESET either.
// In pure RESET each separate URL should lead us directly to what we are looking for - the disadvantage: a lot of http queries.

package main

import (
	// Package json implements encoding and decoding of JSON
	"encoding/json"

	// Package strconv implements conversions to and from string representations of basic data types.
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

	// Package url parses URLs and implements query escaping.

	// Package time provides functionality for measuring and displaying time.
	"time"

	// Package os provides a platform-independent interface to operating system functionality.
	"os"
)

// const CHAT_URL = "https://localhost:8443/chat/messages.json"
const CHAT_URL = "https://jch.irif.fr:8082/chat/messages.json"
const EXIT_FAILURE = 1
const MESSEGES_COUNT = 4

type Response struct {
	Id   string `json:"id"`
	Time int64  `json:"time"`
	Body string `json:"body"`
}

func getHttpResponse(client *http.Client, requestUrl string, lastEtag string) ([]byte, int, string) {
	fmt.Printf("HTTP GET REQUEST : %v \n", requestUrl)

	// func http.NewRequest(method string, url string, body io.Reader) (*http.Request, error)
	req, errorMessage := http.NewRequest("GET", requestUrl, nil)
	if errorMessage != nil {
		log.Printf("http.NewRequest() function : %v", errorMessage)
		// func os.Exit(code int)
		os.Exit(EXIT_FAILURE)
	}

	// func (http.Header).Add(key string, value string)
	if len(lastEtag) != 0 {
		req.Header.Add("If-None-Match", lastEtag)
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

	bodyLen := 0
	// field StatusCode int
	if r.StatusCode != 304 { // 304 : Not Modified
		contentLengthHeader := r.Header.Get("content-length")
		if len(contentLengthHeader) == 0 { // It seems that when there are many messages in the chat there is no content-length in the reply
			bodyLen = len(body)
		} else {
			bodyLen, errorMessage = strconv.Atoi(contentLengthHeader)
			if errorMessage != nil {
				// func log.Printf(format string, v ...any)
				log.Printf("strconv.Atoi function : %v", errorMessage)
				// func os.Exit(code int)
				os.Exit(EXIT_FAILURE)
			}
		}
	}

	return body, bodyLen, r.Header.Get("etag")
}

func main() {
	transport := &*http.DefaultTransport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // This is a code for pedagogical purposes !
	client := &http.Client{
		Transport: transport,
		Timeout:   50 * time.Second,
	}

	// func getHttpResponse(client *http.Client, requestUrl url, lastEtag string) ([]byte, int, string)
	body, lastContentLength, lastEtag := getHttpResponse(client, CHAT_URL, "")
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

	forBeginning := 0
	if len(messages) > MESSEGES_COUNT {
		forBeginning = len(messages) - MESSEGES_COUNT
	}

	for i := forBeginning; i < len(messages); i++ {
		// func fmt.Printf(format string, a ...any) (n int, err error)
		fmt.Printf("Id %v: %v\n", i, messages[i].Body)
	}

	fmt.Printf("\n")

	for true {
		// func time.Sleep(d time.Duration)
		time.Sleep(10 * time.Second)

		body, contentLength, newLastEtag := getHttpResponse(client, CHAT_URL, lastEtag)
		fmt.Printf("TEST lastEtag : %v \n", lastEtag)
		fmt.Printf("TEST newLastEtag : %v \n", newLastEtag)

		fmt.Printf("\n")

		if contentLength > 0 {
			// Exemple : body[lastContentLength:] == "id":"7214f1750498737c4e96a72bb2edf483","time":1665169107721,"body":"new message 4"}]
			// bodyAfterAppend = [{"id":"7214f1750498737c4e96a72bb2edf483","time":1665169107721,"body":"new message 4"}]

			bodyAfterAppend := append([]byte{byte('['), byte('{')}, body[lastContentLength:]...)
			errorMessage := json.Unmarshal(bodyAfterAppend, &messages)

			for _, char := range bodyAfterAppend {
				fmt.Printf("%c", char)
			}

			if errorMessage != nil {
				// func log.Printf(format string, v ...any)
				log.Printf("json.Unmarshal() function : %v \n", errorMessage)
				// func os.Exit(code int)
				os.Exit(EXIT_FAILURE)
			}

			for _, message := range messages {
				// func fmt.Printf(format string, a ...any) (n int, err error)
				fmt.Printf("Id : %v\n", message.Body)
			}

			lastContentLength = contentLength
			lastEtag = newLastEtag

			fmt.Printf("\n")
		}
	}
}
