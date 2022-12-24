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
const NAME_FOR_SERVER_REGISTRATION = "HugoLeonard"

func main() {
	//rand.Seed(int64(time.Now().Nanosecond()))
	var datagram_id = "idid"

	httpClient := CreateHttpClient()

	/* STEP 1 : GET THE UDP ADDRESS OF THE SERVER
	 *  HTTP GET to /udp-address followed by a JSON decode.
	 */
	requestUrl := url.URL{Scheme: "https", Host: HOST, Path: "/udp-address"}
	httpResponseBody, _ := HttpRequest("GET", httpClient, requestUrl.String(), nil)

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
	httpResponseBody, _ = HttpRequest("POST", httpClient, requestUrl.String(), jsonEncoding)

	/* STEP 3 : GET THE SERVER'S PUBLIC KEY
	 * THE PUBLIC KEY THAT THE SERVER USES TO SIGN MESSAGES IS AVAILABLE AT /server-key.
	 * IF A GET TO THIS URL RETURNS 404, THE SERVER DOES NOT SIGN ITS MESSAGES.
	 */
	requestUrl = url.URL{Scheme: "https", Host: HOST, Path: "/server-key"}
	httpResponseBody, _ = HttpRequest("GET", httpClient, requestUrl.String(), nil)

	/* STEP 4 : HELLO TO EACH OF THE ADDRESSES OBTAINED IN STEP 1
	 */

	//myByteId := CreateRandId()

	// func net.Dial(network string, address string) (net.Conn, error)
	conn, errorMessage := net.ListenPacket("udp", fmt.Sprintf(":%d", 8081))
	if errorMessage != nil {
		log.Fatalf("The method net.ListenUDP() failed in sendUdp() to address : %v\n", errorMessage)
	}
	log.Printf("Listening to :%d \n", 8082)
	go UdpRead(conn)

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

		UdpWrite(conn, datagram_id, HELLO_TYPE, serverAddr, nil)
	}

	/* STEP 5 : LIST OF PEERS KNOWN TO THE SERVER
	 * A GET REQUEST TO THE URL /peers.
	 * THE SERVER RESPONDS WITH THE BODY CONTAINING A LIST OF PEER NAMES, ONE PER LINE.
	 */
	requestUrl = url.URL{Scheme: "https", Host: HOST, Path: "/peers"}
	httpResponseBody, _ = HttpRequest("GET", httpClient, requestUrl.String(), nil)

	/* STEP 6 : PEER ADDRESSES
	 * TO LOCATE A PEER NAMED p, THE CLIENT MAKES A GET REQUEST TO THE URL /peers/p.
	 * THE RESPONSE BODY CONTAINS A JSON OBJECT
	 */
	var peers []Peer

	bodyAfterSplit := strings.Split(string(httpResponseBody), "\n")
	for _, p := range bodyAfterSplit {

		if len(p) != 0 {
			peerUrl := requestUrl.String() + "/" + p
			bodyfromPeer, statusCode := HttpRequest("GET", httpClient, peerUrl, nil)

			if statusCode == 200 {
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

	/* STEP 7 : HELLO TO ALL PEER ADDRESSES
	 */
	for _, peer := range peers {
		if peer.Username != NAME_FOR_SERVER_REGISTRATION {
			for _, address := range peer.Adresses {
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

				UdpWrite(conn, datagram_id, HELLO_TYPE, serverAddr, nil)
			}
		}
	}

	/* STEP 8 : ROOT REQUEST TO ALL PEER ADDRESSES
	 */
	for _, peer := range peers {
		if peer.Username != NAME_FOR_SERVER_REGISTRATION {
			for _, address := range peer.Adresses {
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

				UdpWrite(conn, datagram_id, ROOT_REQUEST_TYPE, serverAddr, nil)
			}
		}
	}

	/* STEP 9 :
	 */
	fmt.Println()
	serverAddr, err := net.ResolveUDPAddr("udp", "[2a01:e0a:283:47b0:ba:7bff:fed5:c602]:1111")
	if err != nil {
		panic(err)
	}

	UdpWrite(conn, datagram_id, GET_DATUM_TYPE, serverAddr, []byte{80, 118, 133, 10, 109, 125, 229, 201, 82, 105, 128, 65, 40, 17, 68, 247, 5, 223, 18, 113, 2, 67, 177, 164, 28, 236, 120, 121, 129, 106, 44, 144})

	fmt.Println()
	log.Printf("WAITING FOR NEW MESSAGES... \n")
	for {
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

func CreateRandId() []byte {
	id := new(bytes.Buffer)
	err := binary.Write(id, binary.LittleEndian, rand.Int31())
	if err != nil {
		fmt.Println("binary.Write failed in CreateRandId() :", err)
	}
	return id.Bytes()
}
