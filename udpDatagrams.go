package main

import (
	"fmt"
)

const DATAGRAM_MIN_LENGTH = 4 + 1 + 2 // For Id, Type, Length
const SIGNATURE_LENGTH = 0
const HELLO_DATAGRAM_BODY_MIN_LENGTH = 4 + 1 // For Flags and Username Length

const INTERNAL_NODE_DATAGRAM_MIN_LENGTH = 1     // For the Type field
const LEAF_DATAGRAM_MIN_LENGTH = 1 + 4 + 32 + 2 // For the Type, Date, In-reply-toLength and Length fields

// General structure of a datagram
func datagramGeneralStructure(datagramId []byte, datagramType int, datagramBodyLength int, datagramLength int) []byte {
	datagram := make([]byte, datagramLength)

	copy(datagram[0:4], datagramId)
	datagram[4] = byte(datagramType)
	datagram[5] = byte(datagramBodyLength >> 8)
	datagram[6] = byte(datagramBodyLength & 0xFF)

	return datagram
}

func PrintDatagram(datagram []byte) {
	lengthBody := datagram[5]<<8 + datagram[6]
	//lengthBody := int(datagram[5])<<8 | int(datagram[6])

	fmt.Printf("THE DATAGRAMME AS BYTES : %v \n", datagram[:(DATAGRAM_MIN_LENGTH+lengthBody+SIGNATURE_LENGTH)])
	fmt.Printf("THE DATAGRAMME AS STRING : %s \n", datagram)

	id := datagram[0:4]
	responseType := datagram[4]
	fmt.Printf("ID : %s TYPE : %d LENGTH : %d  \n", id, responseType, lengthBody)

	if lengthBody > 0 {
		body := datagram[12:]
		fmt.Printf("BODY : %s \n", body)
	}
}

/********************************************** MERKEL TREE **********************************************/

func MerkleInternalNodeToUdp(merkleNode *MerkleNode) []byte {
	var datagram_type byte = 1 // Type 1 indicates that this is an internal node.

	datagram_length := INTERNAL_NODE_DATAGRAM_MIN_LENGTH + len(merkleNode.Hash)
	datagram := make([]byte, datagram_length)

	datagram[0] = datagram_type
	copy(datagram[1:], merkleNode.Hash)

	return datagram
}

func LeafToUdp(merkleNode *MerkleNode) []byte {
	var datagram_type byte = 0 // Type 0 indicates that it is a message

	message_body_length := len(merkleNode.message.Body)
	datagram_length := LEAF_DATAGRAM_MIN_LENGTH + message_body_length
	datagram := make([]byte, datagram_length)

	datagram[0] = datagram_type
	copy(datagram[1:4], []byte(merkleNode.message.Date.String())) // warning : signed or nor signed ?
	copy(datagram[4:36], merkleNode.message.InReplyTo)
	copy(datagram[4:36], merkleNode.message.InReplyTo)

	datagram[36] = byte(message_body_length >> 8)
	datagram[37] = byte(message_body_length & 0xFF)

	copy(datagram[LEAF_DATAGRAM_MIN_LENGTH:], merkleNode.message.Body)

	return datagram
}

/********************************************** HELLO, HELLO_REPLY **********************************************/
func HelloDatagram(id string, userName string) []byte {
	usernameLength := len(userName)
	datagramBodyLength := HELLO_DATAGRAM_BODY_MIN_LENGTH + usernameLength
	datagramLength := DATAGRAM_MIN_LENGTH + datagramBodyLength + SIGNATURE_LENGTH

	// Hello messages :  type = 0
	datagram := datagramGeneralStructure([]byte(id), 0, datagramBodyLength, datagramLength)

	copy(datagram[7:11], []byte{0, 0, 0, 0})
	datagram[11] = byte(usernameLength)
	copy(datagram[12:], userName)

	return datagram
}
