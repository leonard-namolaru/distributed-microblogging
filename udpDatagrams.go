package main

import (
	"crypto/sha256"
	"fmt"
)

const DATAGRAM_MAX_LENGTH_IN_BYTES = 1024

const DATAGRAM_MIN_LENGTH = 4 + 1 + 2 // For Id, Type, Length
const SIGNATURE_LENGTH = 0
const HELLO_DATAGRAM_BODY_MIN_LENGTH = 4 + 1 // For Flags and Username Length

const INTERNAL_NODE_DATAGRAM_MIN_LENGTH = 1     // For the Type field
const LEAF_DATAGRAM_MIN_LENGTH = 1 + 4 + 32 + 2 // For the Type, Date, In-reply-toLength and Length fields

const ROOT_BODY_LENGTH = 32
const ROOT_REQUEST_BODY_LENGTH = 0

const GET_DATUM_BODY_LENGTH = 32
const NO_DATUM_BODY_LENGTH = 32

const ID_LENGTH = 4
const ID_FIRST_BYTE = 0

const TYPE_BYTE = 4

const BODY_FIRST_BYTE = 7

const FLAGS_FIRST_BYTE = 7
const FLAGS_LENGTH = 4

const USER_NAME_LENGTH_BYTE = 11
const USER_NAME_FIRST_BYTE = 12

const HASH_LENGTH = 32

// General structure of a datagram
func datagramGeneralStructure(datagramId []byte, datagramType int, datagramBodyLength int, datagramLength int) []byte {
	datagram := make([]byte, datagramLength)

	copy(datagram[0:4], datagramId)
	datagram[4] = byte(datagramType)
	datagram[5] = byte(datagramBodyLength >> 8)
	datagram[6] = byte(datagramBodyLength & 0xFF)

	return datagram
}

func PrintDatagram(isDatagramWeSent bool, address string, datagram []byte) {
	var str string
	str = ""

	bodyLength := int(datagram[5]) + int(datagram[6])
	id := datagram[ID_FIRST_BYTE : ID_FIRST_BYTE+ID_LENGTH]
	datagramType := datagram[TYPE_BYTE]

	//if !isDatagramWeSent {
	//	str += fmt.Sprintf("THE UDP DATAGRAM (from %s) :\n", address)
	//} else {
	//	str += fmt.Sprintf("THE UDP DATAGRAM (to : %s) :\n", address)
	//}

	str += fmt.Sprintf("THE DATAGRAM AS BYTES : %v \n", datagram[:(DATAGRAM_MIN_LENGTH+bodyLength+SIGNATURE_LENGTH)])
	str += fmt.Sprintf("ID : %v TYPE : %d LENGTH : %d  \n", id, datagramType, bodyLength)

	switch datagramType {
	case byte(HELLO_TYPE):
		userNameLength := datagram[USER_NAME_LENGTH_BYTE]
		str += fmt.Sprintf("BODY : Flags : %v Username Length : %d Username : %s \n", datagram[FLAGS_FIRST_BYTE:FLAGS_FIRST_BYTE+FLAGS_LENGTH], userNameLength,
			datagram[USER_NAME_FIRST_BYTE:USER_NAME_FIRST_BYTE+userNameLength])

	case byte(HELLO_REPLAY_TYPE):
		userNameLength := datagram[USER_NAME_LENGTH_BYTE]
		str += fmt.Sprintf("BODY : Flags : %v Username Length : %d Username : %s \n", datagram[FLAGS_FIRST_BYTE:FLAGS_FIRST_BYTE+FLAGS_LENGTH], userNameLength,
			datagram[USER_NAME_FIRST_BYTE:USER_NAME_FIRST_BYTE+userNameLength])

	case byte(ROOT_TYPE):
		str += fmt.Sprintf("BODY : %x \n", datagram[BODY_FIRST_BYTE:BODY_FIRST_BYTE+bodyLength])

	case byte(ERROR_TYPE):
		str += fmt.Sprintf("BODY : %s \n", datagram[BODY_FIRST_BYTE:BODY_FIRST_BYTE+bodyLength])

	case byte(DATUM_TYPE):
		str += fmt.Sprintf("BODY  Hash : %x Value : %s \n", datagram[BODY_FIRST_BYTE:BODY_FIRST_BYTE+HASH_LENGTH], datagram[BODY_FIRST_BYTE+HASH_LENGTH+4*8:])

	case byte(GET_DATUM_TYPE):
		str += fmt.Sprintf("BODY : %x \n", datagram[BODY_FIRST_BYTE:BODY_FIRST_BYTE+bodyLength])

	case byte(NO_DATUM_TYPE):
		str += fmt.Sprintf("BODY : %x \n", datagram[BODY_FIRST_BYTE:BODY_FIRST_BYTE+bodyLength])

	}

	fmt.Print(str)
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

func HelloReplayDatagram(id string, userName string) []byte {
	usernameLength := len(userName)
	datagramBodyLength := HELLO_DATAGRAM_BODY_MIN_LENGTH + usernameLength
	datagramLength := DATAGRAM_MIN_LENGTH + datagramBodyLength + SIGNATURE_LENGTH

	// type = 128
	datagram := datagramGeneralStructure([]byte(id), 128, datagramBodyLength, datagramLength)

	copy(datagram[7:11], []byte{0, 0, 0, 0})
	datagram[11] = byte(usernameLength)
	copy(datagram[12:], userName)

	return datagram
}

/********************************************** ROOT_REQUEST, ROOT **********************************************/
func RootRequestDatagram(id string) []byte {
	datagramLength := DATAGRAM_MIN_LENGTH + ROOT_REQUEST_BODY_LENGTH + SIGNATURE_LENGTH
	datagram := datagramGeneralStructure([]byte(id), ROOT_REQUEST_TYPE, ROOT_REQUEST_BODY_LENGTH, datagramLength)
	return datagram
}

func RootDatagram(id string) []byte {
	datagramLength := DATAGRAM_MIN_LENGTH + ROOT_BODY_LENGTH + SIGNATURE_LENGTH
	datagram := datagramGeneralStructure([]byte(id), ROOT_TYPE, ROOT_BODY_LENGTH, datagramLength)

	hash := sha256.New()
	copy(datagram[BODY_FIRST_BYTE:], hash.Sum([]byte{})) // Temporary solution: we answer with the hash of the empty data
	return datagram
}

/********************************************** DATUM, GET_DATUM, NO_DATUM **********************************************/
func GetDatumDatagram(id string, hash []byte) []byte {
	datagramLength := DATAGRAM_MIN_LENGTH + GET_DATUM_BODY_LENGTH + SIGNATURE_LENGTH
	datagram := datagramGeneralStructure([]byte(id), GET_DATUM_TYPE, GET_DATUM_BODY_LENGTH, datagramLength)

	copy(datagram[BODY_FIRST_BYTE:], hash)
	return datagram
}
