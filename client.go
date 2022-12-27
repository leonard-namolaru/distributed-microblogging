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
const MERKLE_TREE_MAX_ARITY = 32
const UDP_LISTENING_ADDRESS = ":8081"

var ThisPeerMerkleTree = CreateTree(CreateMessagesForMerkleTree(33), MERKLE_TREE_MAX_ARITY)

func main() {
	//rand.Seed(int64(time.Now().Nanosecond()))
	var datagram_id = "idid"

	httpClient := CreateHttpClient()

	/* A LIST OF MESSAGES AVAILABLE TO THE OTHER PEERS
	 */
	ThisPeerMerkleTree.DepthFirstSearch(0, ThisPeerMerkleTree.PrintNodesData, nil)

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
		log.Fatalf("The method json.Marshal() failed at the stage of encoding the JSON object for server registration :  %v \n", err)
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
	conn, errorMessage := net.ListenPacket("udp", UDP_LISTENING_ADDRESS)
	if errorMessage != nil {
		log.Fatalf("The method net.ListenUDP() failed in sendUdp() to address : %v\n", errorMessage)
	}
	log.Printf("LISTENING TO %s \n", UDP_LISTENING_ADDRESS)

	// The reading of the received datagrams is done in a separate thread
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

	/* STEP 7 : HELLO TO (ALL) PEER ADDRESSES
	 */
	for _, peer := range peers {
		if peer.Username == "jch" || peer.Username == "bet" {
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

	/* STEP 8 : ROOT REQUEST TO ALL THE SESSIONS WE OPENED
	 */
	for _, session := range sessionsWeOpened {
		// We also have an open session with the server but we are now interested in contacting only the peers with whom we created a session.
		if session.FullAddress.IP.String() != serverUdpAddresses[0].Ip && session.FullAddress.Port != int(serverUdpAddresses[0].Port) || (session.FullAddress.IP.String() != serverUdpAddresses[1].Ip && session.FullAddress.Port != int(serverUdpAddresses[1].Port)) {
			UdpWrite(conn, datagram_id, ROOT_REQUEST_TYPE, &session.FullAddress, nil)
		}
	}

	/* STEP 9 : USING THE HASH OF THE ROOT IN ORDER TO OBTAIN THE INFORMATION THAT THE HASH REPRESENTS
	 */
	for i := 0; i < len(sessionsWeOpened); i++ {
		if len(sessionsWeOpened[i].buffer) != 0 {
			writeResult := UdpWrite(conn, datagram_id, GET_DATUM_TYPE, &sessionsWeOpened[i].FullAddress, sessionsWeOpened[i].buffer[0])
			if writeResult {

				getDatumResult := getDatum(conn, i, datagram_id)
				if getDatumResult || true {
					var messages [][]byte

					for j := 1; j < len(sessionsWeOpened[i].buffer); j++ {

						if int(sessionsWeOpened[i].buffer[j][HASH_LENGTH+NODE_TYPE_BYTE]) == 0 {
							messages = append(messages, sessionsWeOpened[i].buffer[j][HASH_LENGTH:])
						}
					}

					sessionsWeOpened[i].Merkle = CreateTree(messages, MERKLE_TREE_MAX_ARITY)
					sessionsWeOpened[i].Merkle.DepthFirstSearch(0, ThisPeerMerkleTree.PrintNodesData, nil)
				}
			}
		}

	}

	fmt.Println()
	fmt.Printf("...\n")

	for {
	}

}

func checkHash(hash []byte, data []byte) bool {
	return true
}

func getDatum(conn net.PacketConn, sessionIndex int, datagramId string) bool {
	bufferIndex := len(sessionsWeOpened[sessionIndex].buffer) - 1
	datagramBody := sessionsWeOpened[sessionIndex].buffer[bufferIndex]
	hash := datagramBody[0:HASH_LENGTH]

	if HASH_LENGTH+MESSAGE_BODY_FIRST_BYTE < len(datagramBody) {
		messageLength := int(datagramBody[HASH_LENGTH+LENGTH_FIRST_BYTE])<<8 | int(datagramBody[HASH_LENGTH+LENGTH_FIRST_BYTE+1])
		messageBody := datagramBody[HASH_LENGTH+MESSAGE_BODY_FIRST_BYTE:]
		if datagramBody[HASH_LENGTH+NODE_TYPE_BYTE] == 0 || messageLength == len(messageBody) {
			return true
		}
	}

	if len(datagramBody) == HASH_LENGTH {
		sessionsWeOpened[sessionIndex].buffer = append(sessionsWeOpened[sessionIndex].buffer[:bufferIndex], sessionsWeOpened[sessionIndex].buffer[bufferIndex+1:]...) // We remove the session
		return false
	}

	if !checkHash(hash, datagramBody[HASH_LENGTH:]) {
		return false
	}

	for i := 1 + HASH_LENGTH; i < len(datagramBody); i += HASH_LENGTH {
		hashI := datagramBody[i : i+HASH_LENGTH]

		writeResult := UdpWrite(conn, datagramId, GET_DATUM_TYPE, &sessionsWeOpened[sessionIndex].FullAddress, hashI)
		if writeResult {
			getDatum(conn, sessionIndex, datagramId)
		} else {
			return false
		}
	}

	return true
}

func CreateRandId() []byte {
	id := new(bytes.Buffer)
	err := binary.Write(id, binary.LittleEndian, rand.Int31())
	if err != nil {
		fmt.Println("binary.Write failed in CreateRandId() :", err)
	}
	return id.Bytes()
}

/*
func CreateRandId2() []byte {
	var id [4]byte
	copy(id, rand.Int31())
	return id
}
*/
