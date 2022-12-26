package main

import (
	"fmt"
	"log"
	"time"
)

var JANUARY_1_2022 = time.Date(2022, 1, 1, 1, 0, 0, 0, time.Local) // January 1, 2022

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

const DATUM_VALUE_FIRST_BYTE = BODY_FIRST_BYTE + HASH_LENGTH

const ID_LENGTH = 4
const ID_FIRST_BYTE = 0

const TYPE_BYTE = 4

const BODY_FIRST_BYTE = 7

const FLAGS_FIRST_BYTE = 7
const FLAGS_LENGTH = 4

const USER_NAME_LENGTH_BYTE = 11
const USER_NAME_FIRST_BYTE = 12

const HASH_LENGTH = 32

/* ***** */

const NODE_TYPE_BYTE = 0

const MESSAGE_DATE_FIRST_BYTE = 1
const MESSAGE_DATE_LENGTH = 4
const MESSAGE_IN_REPLY_TO_FIRST_BYTE = 5
const MESSAFE_IN_REPLY_TO_LENGTH = 32
const MESSAFE_LENGTH_FIRST_BYTE = 37
const MESSAGE_LENGTH_LENGTH = 2
const MESSAGE_BODY_FIRST_BYTE = 39
const MESSAGE_TOTAL_MIN_LENGTH = 1 + MESSAGE_DATE_LENGTH + MESSAFE_IN_REPLY_TO_LENGTH + MESSAGE_LENGTH_LENGTH // 1 for the type byte

// General structure of a datagram
func datagramGeneralStructure(datagramId []byte, datagramType int, datagramBodyLength int, datagramLength int) []byte {
	datagram := make([]byte, datagramLength)
	copy(datagram[0:4], datagramId)
	datagram[4] = byte(datagramType)
	datagram[5] = byte(datagramBodyLength >> 8)
	datagram[6] = byte(datagramBodyLength & 0xFF)

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

	copy(datagram[BODY_FIRST_BYTE:], ThisPeerMerkleTree.Root.Hash)
	return datagram
}

/********************************************** DATUM, GET_DATUM, NO_DATUM **********************************************/
func GetDatumDatagram(id string, hash []byte) []byte {
	datagramLength := DATAGRAM_MIN_LENGTH + GET_DATUM_BODY_LENGTH + SIGNATURE_LENGTH
	datagram := datagramGeneralStructure([]byte(id), GET_DATUM_TYPE, GET_DATUM_BODY_LENGTH, datagramLength)

	copy(datagram[BODY_FIRST_BYTE:], hash)
	return datagram
}

func DatumDatagram(id string, hash []byte) []byte {
	node := ThisPeerMerkleTree.DepthFirstSearch(0, ThisPeerMerkleTree.GetNodeByHash, hash)
	if node == nil {
		return NoDatumDatagram(id, hash)
	}

	datagramBodyLength := HASH_LENGTH + len(node.Data)
	datagramLength := DATAGRAM_MIN_LENGTH + datagramBodyLength + SIGNATURE_LENGTH
	datagram := datagramGeneralStructure([]byte(id), DATUM_TYPE, datagramBodyLength, datagramLength)

	copy(datagram[BODY_FIRST_BYTE:BODY_FIRST_BYTE+HASH_LENGTH], node.Hash)
	copy(datagram[DATUM_VALUE_FIRST_BYTE:], node.Data)
	return datagram
}

func NoDatumDatagram(id string, hash []byte) []byte {
	datagramLength := DATAGRAM_MIN_LENGTH + NO_DATUM_BODY_LENGTH + SIGNATURE_LENGTH
	datagram := datagramGeneralStructure([]byte(id), NO_DATUM_TYPE, NO_DATUM_BODY_LENGTH, datagramLength)

	copy(datagram[BODY_FIRST_BYTE:], hash)
	return datagram
}

/********************************************** ERROR **********************************************/
func ErrorDatagram(id string, errorMessage []byte) []byte {
	datagramBodyLength := len(errorMessage)
	datagramLength := DATAGRAM_MIN_LENGTH + datagramBodyLength + SIGNATURE_LENGTH
	datagram := datagramGeneralStructure([]byte(id), ERROR_TYPE, datagramBodyLength, datagramLength)

	copy(datagram[BODY_FIRST_BYTE:], errorMessage)
	return datagram
}

/******************************** DATAGRAM TO STRING / PRINT DATAGRAM **************************************/

func PrintDatagram(isDatagramWeSent bool, address string, datagram []byte, timeOut float64) {
	var str string
	str = ""

	bodyLength := int(datagram[5]) + int(datagram[6])
	id := datagram[ID_FIRST_BYTE : ID_FIRST_BYTE+ID_LENGTH]
	datagramType := datagram[TYPE_BYTE]

	if !isDatagramWeSent {
		str += fmt.Sprintf("WE RECEIVE A DATAGRAM FROM %s :\n", address)
	} else {
		str += fmt.Sprintf("WE SEND A DATAGRAM TO : %s :\n", address)
	}

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
		str += fmt.Sprintf("BODY : %s ", datumDatagramToString(datagram[BODY_FIRST_BYTE:BODY_FIRST_BYTE+bodyLength]))
	case byte(GET_DATUM_TYPE):
		str += fmt.Sprintf("BODY : %x \n", datagram[BODY_FIRST_BYTE:BODY_FIRST_BYTE+bodyLength])
	case byte(NO_DATUM_TYPE):
		str += fmt.Sprintf("BODY : %x \n", datagram[BODY_FIRST_BYTE:BODY_FIRST_BYTE+bodyLength])
	}

	if timeOut > 0 {
		str += fmt.Sprintf("TIMEOUT AFTER %.2f SEC \n", timeOut)
	}

	fmt.Println()
	log.Print(str)
}

func datumDatagramToString(datumDatagramBody []byte) string {
	var str string
	Hash := datumDatagramBody[0:HASH_LENGTH]

	str = fmt.Sprintf("Node hash : %x \n", Hash)
	str += nodeDataToString(datumDatagramBody[HASH_LENGTH:], 0)
	return str
}
