// Installation (UBUNTU) : go get github.com/gorilla/websocket
// The source of the comments: the official documentation of GO

package main

import (
	// Package tls partially implements TLS 1.2, as specified in RFC 5246, and TLS 1.3, as specified in RFC 8446.
	"crypto/tls"
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

	get := &jsonMessage{Type: "get", Count: 20}

	//func (c *Conn) WriteJSON(v interface{}) error
	// WriteJSON writes the JSON encoding of v as a message.
	// See the documentation for encoding/json Marshal for details about the conversion of Go values to JSON.
	errorMessage = connection.WriteJSON(&get)
	if errorMessage != nil {
		log.Fatal("Function WriteJSON(): ", errorMessage)
	}

	var response jsonMessage

	// func (c *Conn) ReadJSON(v interface{}) error
	// ReadJSON reads the next JSON-encoded message from the connection and stores it in the value pointed to by v.
	// See the documentation for the encoding/json Unmarshal function for details about the conversion of JSON to a Go value.
	errorMessage = connection.ReadJSON(&response)
	if errorMessage != nil {
		log.Fatal("Function ReadJSON(): ", errorMessage)
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
