package main

import (
	"time"
)

var JANUARY_1_2022 = time.Date(2022, 1, 1, 1, 0, 0, 0, time.Local) // January 1, 2022

type Message struct {
	Date      time.Duration // Encoded as a number of seconds since January 1, 2022
	InReplyTo []byte        // The hash of the message to which this message replies, or 0
	Body      string        // The message itself, encoded in UTF-8
}

func CreateMicroBloggingMessage(body string, inReplayTo []byte) *Message {

	date := time.Since(JANUARY_1_2022)

	return &Message{Date: date, InReplyTo: inReplayTo, Body: body}
}

func main() { // For testing purposes only
}
