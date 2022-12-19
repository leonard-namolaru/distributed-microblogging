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
	copy(datagram[6:38],[]byte(receivedHasedMessage))
	datagram[39] = byte(datagram_body_length >> 8)
	datagram[40] = byte(datagram_body_length & 0xFF)
	copy(datagram[41:], []byte(sendedMessage))

	return datagram
}

func CreateHello(id []byte) []byte { // signature not implemanted
	datagramLength := 12+len(myId.Name) // if signature are implemanted that's more
	datagramBodyLength := datagramLength-7
	datagram := make([]byte, datagramLength)
	copy(datagram[0:3],id)
	datagram[4] = 0
	datagram[5] = byte(datagramBodyLength >> 8)
	datagram[6] = byte(datagramBodyLength & 0xFF)
	copy(datagram[7:10],([]byte)(0)) //recently we don't have implemant extention
	datagram[11] = len(myId.Name)
	copy(datagram[12:12+len(myId.Name)],([]byte)(myId.Name))
	
	return datagram
}

func CreateRandId() []byte {
	var id [4]byte
	copy(id, rand.Int31())
	return id
}