package main

import (
	"crypto/ecdsa"
	"fmt"
	"log"
)

/* Datagram types */
const HELLO_TYPE = 0
const ROOT_REQUEST_TYPE = 1
const GET_DATUM_TYPE = 2
const HELLO_REPLY_TYPE = 128
const ROOT_TYPE = 129
const DATUM_TYPE = 130
const NO_DATUM_TYPE = 131
const ERROR_TYPE = 254

/* General structure of a datagram */
const DATAGRAM_MIN_LENGTH = 4 + 1 + 2 // For Id, Type, Length
const ID_FIRST_BYTE = 0
const ID_LENGTH = 4
const TYPE_BYTE = 4
const LENGTH_FIRST_BYTE = 5
const BODY_FIRST_BYTE = 7

/* Hello, HelloReply */
const HELLO_DATAGRAM_BODY_MIN_LENGTH = 4 + 1 // For Flags and Username Length
const FLAGS_FIRST_BYTE = 7
const FLAGS_LENGTH = 4
const USER_NAME_LENGTH_BYTE = 11
const USER_NAME_FIRST_BYTE = 12

const ROOT_BODY_LENGTH = 32
const ROOT_REQUEST_BODY_LENGTH = 0
const GET_DATUM_BODY_LENGTH = 32
const NO_DATUM_BODY_LENGTH = 32
const DATUM_VALUE_FIRST_BYTE = BODY_FIRST_BYTE + HASH_LENGTH

const HASH_LENGTH = 32
const SIGNATURE_LENGTH = 64

/* General structure of a datagram
Each datagram includes the Id, Type and Length fields before the Body.
n order not to repeat these definitions in every function that handles the construction
of a particular datagram, these definitions are made in this function.
*/
func datagramGeneralStructure(datagramId []byte, datagramType int, datagramBodyLength int, datagramLength int) []byte {
	datagram := make([]byte, datagramLength)
	copy(datagram[ID_FIRST_BYTE:ID_FIRST_BYTE+ID_LENGTH], datagramId)
	datagram[TYPE_BYTE] = byte(datagramType)
	datagram[LENGTH_FIRST_BYTE] = byte(datagramBodyLength >> 8)     // Shift the higher 8 bits
	datagram[LENGTH_FIRST_BYTE+1] = byte(datagramBodyLength & 0xFF) // Mask the lower 8 bits

	return datagram
}

/********************************************** HELLO, HELLO_REPLY **********************************************/
/*
Since the structures of the Hello and HelloReply datagrams are completely identical (except for the byte of the datagram type),
the construction of both is done using a single function. When the parameter isHelloDatagram is true, the function returns a datagram of type Hello.
Otherwise, the function returns a datagram of type HelloReply.
*/
func HelloOrHelloReplyDatagram(isHelloDatagram bool, id string, userName string, privateKey *ecdsa.PrivateKey) []byte {
	usernameLength := len(userName)
	datagramBodyLength := HELLO_DATAGRAM_BODY_MIN_LENGTH + usernameLength
	datagramLength := DATAGRAM_MIN_LENGTH + datagramBodyLength + SIGNATURE_LENGTH
	datagramType := HELLO_TYPE
	if !isHelloDatagram {
		datagramType = HELLO_REPLY_TYPE
	}
	datagram := datagramGeneralStructure([]byte(id), datagramType, datagramBodyLength, datagramLength)

	copy(datagram[FLAGS_FIRST_BYTE:FLAGS_FIRST_BYTE+FLAGS_LENGTH], []byte{0, 0, 0, 0})
	datagram[USER_NAME_LENGTH_BYTE] = byte(usernameLength)
	copy(datagram[USER_NAME_FIRST_BYTE:USER_NAME_FIRST_BYTE+usernameLength], userName)

	datagramWithSignature := CreateSignature(datagram, datagramLength, privateKey)

	return datagramWithSignature
}

/********************************************** ROOT_REQUEST, ROOT **********************************************/
func RootRequestDatagram(id string, privateKey *ecdsa.PrivateKey) []byte {
	datagramLength := DATAGRAM_MIN_LENGTH + ROOT_REQUEST_BODY_LENGTH + SIGNATURE_LENGTH
	datagram := datagramGeneralStructure([]byte(id), ROOT_REQUEST_TYPE, ROOT_REQUEST_BODY_LENGTH, datagramLength)

	datagramWithSignature := CreateSignature(datagram, datagramLength, privateKey)

	return datagramWithSignature
}

func RootDatagram(id string, privateKey *ecdsa.PrivateKey) []byte {
	datagramLength := DATAGRAM_MIN_LENGTH + ROOT_BODY_LENGTH + SIGNATURE_LENGTH
	datagram := datagramGeneralStructure([]byte(id), ROOT_TYPE, ROOT_BODY_LENGTH, datagramLength)

	copy(datagram[BODY_FIRST_BYTE:], ThisPeerMerkleTree.Root.Hash)

	datagramWithSignature := CreateSignature(datagram, datagramLength, privateKey)

	return datagramWithSignature
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
	bodyLength := int(datagram[LENGTH_FIRST_BYTE])<<8 | int(datagram[LENGTH_FIRST_BYTE+1])
	id := datagram[ID_FIRST_BYTE : ID_FIRST_BYTE+ID_LENGTH]
	datagramType := datagram[TYPE_BYTE]

	if !isDatagramWeSent {
		str += fmt.Sprintf("WE RECEIVE A DATAGRAM FROM %s :\n", address)
	} else {
		str += fmt.Sprintf("WE SEND A DATAGRAM TO : %s :\n", address)
	}

	str += fmt.Sprintf("THE DATAGRAM AS BYTES : %v \n", datagram[:(DATAGRAM_MIN_LENGTH+bodyLength+SIGNATURE_LENGTH)])
	str += fmt.Sprintf("ID : %v TYPE : %d LENGTH : %d  \n", id, datagramType, bodyLength)

	if len(datagram[BODY_FIRST_BYTE:]) > bodyLength { // If there is a signature after the body
		str += fmt.Sprintf("SIGNATURE : %x  \n", datagram[:(DATAGRAM_MIN_LENGTH+bodyLength)])
	}

	switch datagramType {
	case byte(HELLO_TYPE):
		userNameLength := datagram[USER_NAME_LENGTH_BYTE]
		str += fmt.Sprintf("BODY : Flags : %v Username Length : %d Username : %s \n", datagram[FLAGS_FIRST_BYTE:FLAGS_FIRST_BYTE+FLAGS_LENGTH], userNameLength,
			datagram[USER_NAME_FIRST_BYTE:USER_NAME_FIRST_BYTE+userNameLength])

	case byte(HELLO_REPLY_TYPE):
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
	hash := datumDatagramBody[0:HASH_LENGTH]

	str = fmt.Sprintf("Node hash : %x \n", hash)
	str += nodeDataToString(datumDatagramBody[HASH_LENGTH:], 0)
	return str
}
