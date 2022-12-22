package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type ServerRegistration struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type PeerId struct {
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

func main() {
	//rand.Seed(int64(time.Now().Nanosecond()))
	var mypeers []PeerId

	urlPublicKey := url.URL{Scheme: "https", Host: HOST, Path: "/server-key"}
	urlList := url.URL{Scheme: "https", Host: HOST, Path: "/peers"}

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
	serverRegistration := ServerRegistration{Name: "Hugo and Leonard", Key: ""}
	jsonEncoding, err := json.Marshal(serverRegistration)
	if err != nil {
		fmt.Println("The method json.Marshal() failed at the stage of encoding the JSON object for server registration :  %v \n", err)
	}

	requestUrl = url.URL{Scheme: "https", Host: HOST, Path: "/register"}
	httpResponseBody = HttpRequest("POST", httpClient, requestUrl.String(), jsonEncoding)

	//Step 3 : to get public key of server 	OK
	httpResponseBody = HttpRequest("GET", httpClient, urlPublicKey.String(), nil)

	if DEBUG_MODE {
		fmt.Println("BEGIN DEBUG STEP 3")
		fmt.Printf("\tpublic key response : %v", string(httpResponseBody))
		fmt.Println("END DEBUG STEP 3\n")
	}
	//the server don't sign their messages recently

	// Step 4 : Hello and HelloReply with the same id 	OK BUT WITHOUT net.Listen now

	if DEBUG_MODE {
		fmt.Println("BEGIN DEBUG STEP 4")
	}

	//myByteId := CreateRandId()
	myHello := CreateHello(serverRegistration.Name)

	serverAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%v:%v", serverUdpAddresses[0].Ip, serverUdpAddresses[0].Port))
	if err != nil {
		panic(err)
	}

	connServer, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		panic(err)
	}

	if DEBUG_MODE {
		fmt.Printf("\tListen at %v\n", serverAddr.String())
	}

	_, err = connServer.Write(myHello)
	if err != nil {
		fmt.Printf("Response err %v", err)
	}

	response := make([]byte, 1500)
	readerConnection := bufio.NewReader(connServer)
	_, err = readerConnection.Read(response)
	if err != nil {
		panic("read")
	}

	if DEBUG_MODE {
		decryptResponse(response)
	}

	helloReply := CreateHelloReply(response)

	_, err = connServer.Write(helloReply)
	if err != nil {
		fmt.Printf("\tResponse err %v", err)
	}

	readerConnection = bufio.NewReader(connServer)
	_, err = readerConnection.Read(response)
	if err != nil {
		panic("read")
	}

	if DEBUG_MODE {
		decryptResponse(response)
		fmt.Println("END DEBUG STEP 4\n")
	}

	//Step 5 : to get adress pair 	NON OK

	httpResponseBody = HttpRequest("GET", httpClient, urlList.String(), nil)

	if DEBUG_MODE {
		fmt.Println("BEGIN DEBUG STEP 5")
	}

	bodySplit := strings.Split(string(httpResponseBody), "\n")
	for i, p := range bodySplit {
		if DEBUG_MODE {
			fmt.Printf("\t%d,%s\n", i, p)
		}

		urlPeer := urlList.String() + "/" + p
		bodyfromPeer := HttpRequest("GET", httpClient, urlPeer, nil)
		fmt.Printf("\tbody from peer : %s\n", bodyfromPeer)
		var mypeer PeerId
		err := json.Unmarshal(bodyfromPeer, &mypeer)
		if err != nil {
			//log.Fatalf("json.Unmarshal() : %v\n", err)
		}

		mypeers = append(mypeers, mypeer)

		if DEBUG_MODE {
			fmt.Printf("\tmy peer username : %s\n", mypeer.Username)
		}

	}

	if DEBUG_MODE {
		fmt.Println("END DEBUG STEP 5 \n")
	}

}

func createPeerId(username string, addressesPeer []Address, publicKey string) *PeerId {
	peer := &PeerId{
		Username: username,
		Adresses: addressesPeer,
		Key:      publicKey,
	}
	return peer
}

func postFormHttpResponse(client *http.Client, url string, data url.Values) []byte {

	resp, err := client.PostForm(url, data)
	if err != nil {
		log.Fatalf("client.Do() function : %v\n", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("io.ReadAll() function : %v", err)
	}

	defer resp.Body.Close()

	return body
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

	length := int(datagram[5])<<8 | int(datagram[6])

	body := datagram[7 : 7+length]

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
