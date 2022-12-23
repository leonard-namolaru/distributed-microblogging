package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/url"
	"strings"
)

type ServerRegistration struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type Peer struct {
	Username string    `json:"name"`
	Adresses []Address `json:"addresses"`
	Key      string    `json:"key"`
}

type Address struct {
	Ip   string `json:"ip"`
	Port uint64 `json:"port"`
}

const DEBUG_MODE = true
const HOST = "jch.irif.fr:8443"
const NAME_FOR_SERVER_REGISTRATION = "Hugo and Leonard"

func main() {
	//rand.Seed(int64(time.Now().Nanosecond()))
	httpClient := CreateHttpClient()

	/* STEP 1 : GET THE UDP ADDRESS OF THE SERVER
	 *  HTTP GET to /udp-address followed by a JSON decode.
	 */
	requestUrl := url.URL{Scheme: "https", Host: HOST, Path: "/udp-address"}
	httpResponseBody := HttpRequest("GET", httpClient, requestUrl.String(), nil)

	var serverUdpAddresses []Address
	errorMessage := json.Unmarshal(httpResponseBody, &serverUdpAddresses)
	if errorMessage != nil {
		log.Fatalf("The method json.Unmarshal() failed at the stage of decoding the UDP addresses of the server : %v \n", errorMessage)
	}

	if DEBUG_MODE {
		fmt.Println()
		for i, address := range serverUdpAddresses {
			fmt.Printf("%d : address : %s, port : %d \n", i+1, address.Ip, address.Port)
		}
	}

	/* STEP 2 : SERVER REGISTRATION
	 *  A POST REQUEST TO /register
	 */
	serverRegistration := ServerRegistration{Name: NAME_FOR_SERVER_REGISTRATION, Key: ""}
	jsonEncoding, err := json.Marshal(serverRegistration)
	if err != nil {
		fmt.Println("The method json.Marshal() failed at the stage of encoding the JSON object for server registration :  %v \n", err)
	}

	requestUrl = url.URL{Scheme: "https", Host: HOST, Path: "/register"}
	httpResponseBody = HttpRequest("POST", httpClient, requestUrl.String(), jsonEncoding)

	/* STEP 3 : GET THE SERVER'S PUBLIC KEY
	 * THE PUBLIC KEY THAT THE SERVER USES TO SIGN MESSAGES IS AVAILABLE AT /server-key.
	 * IF A GET TO THIS URL RETURNS 404, THE SERVER DOES NOT SIGN ITS MESSAGES.
	 */
	requestUrl = url.URL{Scheme: "https", Host: HOST, Path: "/server-key"}
	httpResponseBody = HttpRequest("GET", httpClient, requestUrl.String(), nil)

	/* STEP 4 : HELLO TO EACH OF THE ADDRESSES OBTAINED IN STEP 1
	 * ok but without net.Listen
	 */

	//myByteId := CreateRandId()
	myHello := CreateHello(serverRegistration.Name)

	for _, address := range serverUdpAddresses {
		var full_address string
		if net.ParseIP(address.Ip).To4() == nil {
			full_address = fmt.Sprintf("[%v]:%v", address.Ip, address.Port)

		} else {
			full_address = fmt.Sprintf("%v:%v", address.Ip, address.Port)
		}
		serverAddr, err := net.ResolveUDPAddr("udp", full_address)
		if err != nil {
			panic(err)
		}

		buffer := sendUdp(myHello, serverAddr)

		if DEBUG_MODE {
			decryptResponse(buffer)
		}
	}

	/* STEP 5 : LIST OF PEERS KNOWN TO THE SERVER
	 * A GET REQUEST TO THE URL /peers.
	 * THE SERVER RESPONDS WITH THE BODY CONTAINING A LIST OF PEER NAMES, ONE PER LINE.
	 */
	requestUrl = url.URL{Scheme: "https", Host: HOST, Path: "/peers"}
	httpResponseBody = HttpRequest("GET", httpClient, requestUrl.String(), nil)

	/* STEP 6 : PEER ADDRESSES
	 * TO LOCATE A PEER NAMED p, THE CLIENT MAKES A GET REQUEST TO THE URL /peers/p.
	 * THE RESPONSE BODY CONTAINS A JSON OBJECT
	 */
	var peers []Peer

	bodyAfterSplit := strings.Split(string(httpResponseBody), "\n")
	for _, p := range bodyAfterSplit {

		if len(p) != 0 {
			peerUrl := requestUrl.String() + "/" + p
			bodyfromPeer := HttpRequest("GET", httpClient, peerUrl, nil)

			var peer Peer
			err := json.Unmarshal(bodyfromPeer, &peer)
			if err != nil {
				log.Fatalf("The method json.Unmarshal() failed at the stage of decoding the json object received as an answer from %s : %v\n", peerUrl, err)
			}

			peers = append(peers, peer)

			if DEBUG_MODE {
				fmt.Printf("Peer key : %s\n", peer.Key)

				for i, address := range peer.Adresses {
					fmt.Printf("Peer ip %d : %s\n", i+1, address.Ip)
					fmt.Printf("Peer port %d : %d\n", i+1, address.Port)
				}
			}
		}
	}

}

func createPeer(username string, addressesPeer []Address, publicKey string) *Peer {
	peer := &Peer{
		Username: username,
		Adresses: addressesPeer,
		Key:      publicKey,
	}
	return peer
}

func CreateHello(id string) []byte { // signature not implemanted

	idLength := 4
	typeLength := 1
	lengthLength := 2
	flagsLength := 4
	usernameLengthLength := 1
	usernameLength := len(id)
	signatureLength := 0

	datagramBodyLength := flagsLength + usernameLengthLength + usernameLength + signatureLength
	datagramLength := idLength + typeLength + lengthLength + datagramBodyLength

	datagram := make([]byte, datagramLength)

	copy(datagram[0:3], id)
	datagram[4] = 0
	datagram[5] = byte(datagramBodyLength >> 8)
	datagram[6] = byte(datagramBodyLength & 0xFF)
	datagram[7] = 0 //recently we don't have implemant extention
	datagram[8] = 0
	datagram[9] = 0
	datagram[10] = 0
	datagram[11] = byte(usernameLength)
	copy(datagram[12:], id)
	//copy(datagram[14+usernameLength:], myId.Name)

	//length := int(datagram[5])<<8 | int(datagram[6])

	//body := datagram[7 : 7+length]
	/*
		if DEBUG_MODE {
			fmt.Println("\tDEBUT DEBUG HELLO")
			fmt.Printf("\t\ttaille de username : %d\n", usernameLength)
			fmt.Printf("\t\ttaille datagram: %d\n", len(datagram))
			fmt.Printf("\t\ttaille datagramBodyLength: %d\n", datagramLength)
			fmt.Printf("\t\ttaille body: %d\n", datagramBodyLength)
			fmt.Printf("\t\ttaille body reel: %d\n", len(datagram[idLength+typeLength+lengthLength:]))
			if len(body) < 5 {
				fmt.Printf("len(body) = %d\n", len(body))
			}
			fmt.Println("\tFIN DEBUG HELLO\n")
		}
	*/
	return datagram
}

func CreateHelloReply(response []byte) []byte {
	datagram := make([]byte, len(response))
	copy(datagram[0:3], response[0:3])
	datagram[4] = 128
	copy(datagram[5:], response[5:])
	return datagram
}

func decryptResponse(response []byte) {
	id := response[0:3]
	typeResponse := response[4]
	lengthBody := response[5]<<8 + response[6]
	fmt.Printf("\tId : %s\n", id)
	fmt.Printf("\ttype : %s\n", typeResponse)
	fmt.Printf("\ttaille body : %s\n", lengthBody)
}

func CreateRandId() []byte {
	id := new(bytes.Buffer)
	err := binary.Write(id, binary.LittleEndian, rand.Int31())
	if err != nil {
		fmt.Println("binary.Write failed in CreateRandId() :", err)
	}
	return id.Bytes()
}
