package main

import (
	"fmt"
	"time"
)

begin := time.Date(2022, 1, 1, 1, 0, 0, 0, time.Local) // first janvier 2022

func CreateMessage(sendedMessage string,receivedHasedMessage string) []byte { //warning hash or no hash ?

	datagramBody := len(sendedMessage)
	datagram := make([]byte, 1+4+32+datagramBody)
	datagram[0] = 0
	duration := time.Since(begin)
	copy(datagram[1:5],duration.MarshalBinary()) // warning : signed or nor signed ?
	copy(datagram,[6:38]byte(receivedHasedMessage))
	datagram[39] = byte(datagram_body_length >> 8)
	datagram[40] = byte(datagram_body_length & 0xFF)
	copy(datagram[1:], []byte(sendedMessage))

	return datagram
}