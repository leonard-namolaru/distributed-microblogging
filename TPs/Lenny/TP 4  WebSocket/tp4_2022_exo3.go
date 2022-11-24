// Installation (UBUNTU) : go get github.com/gorilla/websocket
// The source of the comments: the official documentation of GO

package main

import (
	// Package tls partially implements TLS 1.2, as specified in RFC 5246, and TLS 1.3, as specified in RFC 8446.
	"crypto/tls"
	"encoding/json"
	"fmt"

	// Package log implements a simple logging package.
	"log"

	// Package url parses URLs and implements query escaping.
	"net/url"

	// Gorilla WebSocket
	// Gorilla WebSocket is a Go implementation of the WebSocket protocol.
	"github.com/gorilla/websocket"
)

type jsonMessage struct {
	Type     string        `json:"type"`
	Message  string        `json:"message,omitempty"`
	Messages []chatMessage `json:"messages,omitempty"`
	Count    int           `json:"count,omitempty"`
	Error    string        `json:"error,omitempty"`
}

type chatMessage struct {
	Id   string `json:"id,omitempty"`
	Time int64  `json:"time,omitempty"`
	Body string `json:"body"`
}

func main() {
	// A URL represents a parsed URL (technically, a URI reference).
	host_url := url.URL{Scheme: "wss", Host: "jch.irif.fr:8443", Path: "/chat/ws"}
	log.Printf("Connecting to %s", host_url.String())

	// This is a code for pedagogical purposes !
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	// func (d *Dialer) Dial(urlStr string, requestHeader http.Header) (*Conn, *http.Response, error)
	// Dial creates a new client connection by calling DialContext with a background context.
	connection, _, errorMessage := dialer.Dial(host_url.String(), nil)
	if errorMessage != nil {
		log.Fatal("Function Dial(): ", errorMessage)
	}

	log.Printf("Connected !")

	errorMessage = connection.WriteMessage(1, []byte(`{"type": "get", "count": 20}`)) // TextMessage = 1
	if errorMessage != nil {
		log.Fatal("Function WriteMessage() :", errorMessage)
	}

	// func (c *Conn) ReadMessage() (messageType int, p []byte, err error)
	// ReadMessage is a helper method for getting a reader using NextReader and reading from that reader to a buffer.
	_, message, errorMessage := connection.ReadMessage()

	if errorMessage != nil {
		// func log.Fatal(v ...any)
		// Fatal is equivalent to Print() followed by a call to os.Exit(1).
		log.Fatal("Function ReadMessage(): %v \n", errorMessage)
	}

	var response jsonMessage

	// func json.Unmarshal(data []byte, v any) error
	// Unmarshal parses the JSON-encoded data and stores the result in the value pointed to by v
	errorMessage = json.Unmarshal(message, &response)
	if errorMessage != nil {
		log.Fatal("json.Unmarshal() function : %v \n", errorMessage)
	}

	for i := 0; i < len(response.Messages); i++ {
		// func fmt.Printf(format string, a ...any) (n int, err error)
		fmt.Printf("%v\n", response.Messages[i].Body)
	}

	// func (c *Conn) Close() error
	// Close closes the underlying network connection without sending or waiting for a close message.
	errorMessage = connection.Close()
	if errorMessage != nil {
		log.Fatal("Function Close(): ", errorMessage)
	}
}
