package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	cryptoRand "crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
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
const NAME_FILE_PRIVATE_KEY = NAME_FOR_SERVER_REGISTRATION + "_key.priv"
const MERKLE_TREE_MAX_ARITY = 32
const UDP_LISTENING_ADDRESS = ":8081"

var peers []Peer
var ThisPeerMerkleTree = CreateTree(CreateMessagesForMerkleTree(33), MERKLE_TREE_MAX_ARITY)

func main() {
	//rand.Seed(int64(time.Now().Nanosecond()))
	var datagram_id = "idid"

	httpClient := CreateHttpClient()

	/* A LIST OF MESSAGES AVAILABLE FOR THE OTHER PEERS
	 */
	ThisPeerMerkleTree.DepthFirstSearch(0, ThisPeerMerkleTree.PrintNodesData, nil)

	/* KEY CRYPTOGRAPHY
	 */

	fileInfo, err := os.Stat(NAME_FILE_PRIVATE_KEY)
	if err != nil || fileInfo.Size() == 0 {

		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), cryptoRand.Reader)
		privateKeyDr, err := x509.MarshalECPrivateKey(privateKey)
		if err != nil {
			panic(err)
		}
		privPEM := pem.EncodeToMemory(
			&pem.Block{
				Type:  "EC PRIVATE KEY",
				Bytes: privateKeyDr,
			},
		)

		err = ioutil.WriteFile(NAME_FILE_PRIVATE_KEY, privPEM, 0644)
		if err != nil {
			panic(err)
		}
	}

	data, err := ioutil.ReadFile(NAME_FILE_PRIVATE_KEY)
	if err != nil {
		panic(err)
	}

	lines := strings.Split(string(data), "\n")
	lines = lines[1 : len(lines)-2]

	privateKeyString := strings.Join(lines, "")
	if DEBUG_MODE {
		fmt.Printf("privateKeyString : %v\n", privateKeyString)
	}

	// Create the private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), bytes.NewReader([]byte(privateKeyString)))
	if err != nil {
		panic(err)
	}

	publicKey, _ := privateKey.Public().(*ecdsa.PublicKey)
	publicKey64Bytes := make([]byte, 64)
	publicKey.X.FillBytes(publicKey64Bytes[:32])
	publicKey.Y.FillBytes(publicKey64Bytes[32:])
	publicKeyEncoded := base64.RawStdEncoding.EncodeToString(publicKey64Bytes)

	if DEBUG_MODE {
		fmt.Printf("Our public key : %s\n", publicKeyEncoded)
	}

	/* GET THE UDP ADDRESS OF THE SERVER
	 *  HTTP GET to /udp-address followed by a JSON decode.
	 */
	requestUrl := url.URL{Scheme: "https", Host: HOST, Path: "/udp-address"}
	httpResponseBody, _ := HttpRequest("GET", httpClient, requestUrl.String(), nil, "%s")

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

	/* SERVER REGISTRATION
	 *  A POST REQUEST TO /register
	 */
	serverRegistration := ServerRegistration{Name: NAME_FOR_SERVER_REGISTRATION, Key: publicKeyEncoded}
	jsonEncoding, err := json.Marshal(serverRegistration)
	if err != nil {
		log.Fatalf("The method json.Marshal() failed at the stage of encoding the JSON object for server registration :  %v \n", err)
	}

	requestUrl = url.URL{Scheme: "https", Host: HOST, Path: "/register"}
	httpResponseBody, _ = HttpRequest("POST", httpClient, requestUrl.String(), jsonEncoding, "%s")

	/* GET THE SERVER'S PUBLIC KEY
	 * THE PUBLIC KEY THAT THE SERVER USES TO SIGN MESSAGES IS AVAILABLE AT /server-key.
	 * IF A GET TO THIS URL RETURNS 404, THE SERVER DOES NOT SIGN ITS MESSAGES.
	 */
	requestUrl = url.URL{Scheme: "https", Host: HOST, Path: "/server-key"}
	publicKeyFromServerBytes, _ := HttpRequest("GET", httpClient, requestUrl.String(), nil, "%x")
	publicKeyFromServerString := base64.RawStdEncoding.EncodeToString(publicKeyFromServerBytes)

	if DEBUG_MODE {
		fmt.Printf("Public Key from server as string : %s\n", publicKeyFromServerString)
	}

	/* HELLO TO EACH OF THE UDP ADDRESSES OF THE SERVER
	 */

	//myByteId := CreateRandId()

	// func net.ListenPacket(network string, address string) (net.PacketConn, error)
	conn, errorMessage := net.ListenPacket("udp", UDP_LISTENING_ADDRESS)
	if errorMessage != nil {
		log.Fatalf("The method net.ListenPacket() failed with %s address : %v\n", UDP_LISTENING_ADDRESS, errorMessage)
	}

	fmt.Println()
	log.Printf("LISTENING TO %s \n", UDP_LISTENING_ADDRESS)

	// The reading of the received datagrams is done in a separate thread
	go UdpRead(conn, privateKey)

	for _, address := range serverUdpAddresses {
		var full_address string
		if net.ParseIP(address.Ip).To4() == nil { // If the address cannot be converted to ipV4
			full_address = fmt.Sprintf("[%v]:%v", address.Ip, address.Port) // ipV6

		} else {
			full_address = fmt.Sprintf("%v:%v", address.Ip, address.Port) // ipV4
		}
		serverAddr, err := net.ResolveUDPAddr("udp", full_address)
		if err != nil {
			log.Fatalf("The method net.ResolveUDPAddr() failed with %s address : %v\n", full_address, errorMessage)
		}

		UdpWrite(conn, datagram_id, HELLO_TYPE, serverAddr, nil, privateKey)
	}

	fmt.Println()
	fmt.Printf("WAITING FOR NEW MESSAGES ...\n")

	var choise string
	var peersKnownToServer []byte = nil

	fmt.Println()
	printMenu()
	for {
		fmt.Scanln(&choise)
		switch choise[0] {
		case 'a':
			fmt.Println()
			fmt.Println("LIST OF PEERS KNOWN TO THE SERVER : ")
			peersKnownToServer = getListPeersKnownToServer(httpClient)
		case 'b':
			var peerName string
			fmt.Println()
			fmt.Println("PEER ADDRESSES : ")
			fmt.Println("Enter peer name : ")
			fmt.Scanln(&peerName)
			if !getPeerAddresses(httpClient, peersKnownToServer, peerName) {
				fmt.Printf("The addresses of the peer %s could not be obtained \n", peerName)
			}
		case 'c':
			var peerAddress string
			fmt.Println()
			fmt.Println("SEND HELLO TO PEER ADDRESS : ")
			fmt.Println("Enter peer address : ")
			fmt.Scanln(&peerAddress)
			if !helloToPeerAddress(conn, peerAddress, datagram_id, privateKey) {
				fmt.Printf("The address %s you specified was not found in the list of addresses of the peers known to the client \n", peerAddress)
			}

		case 'd':
			var peerAddress string
			fmt.Println()
			fmt.Println("ROOT REQUEST TO A OPENED SESSION : ")
			fmt.Println("Enter peer address : ")
			fmt.Scanln(&peerAddress)
			if !rootRequestToOpenedSession(conn, peerAddress, datagram_id, privateKey) {
				fmt.Printf("The address %s you specified was not found in the list of addresses of the opened sessions \n", peerAddress)
			}
		case 'e':
			var peerAddress string
			fmt.Println()
			fmt.Println("OBTAIN THE MERKLE TREE FROM ANOTHER PEER WHO GAVE US THE HASH OF ROOT : ")
			fmt.Println("Enter peer address : ")
			fmt.Scanln(&peerAddress)
			if !getMerkleTreeAnotherPeer(conn, peerAddress, datagram_id, privateKey) {
				fmt.Printf("The address %s you specified was not found in the list of addresses of the opened sessions or we don't have the hash of the root  \n", peerAddress)
			}
		case 'f':
			os.Exit(0)
		default:
			fmt.Println()
			fmt.Printf("Invalid command  \n")
			printMenu()
		}
	}

}

func printMenu() {
	fmt.Println("----- MENU -----")
	fmt.Println("a - List of peers known to the server")
	fmt.Println("b - Get peer addresses")
	fmt.Println("c - Send Hello to peer address")
	fmt.Println("e - Obtain the merkle tree from another peer who gave us the hash of root")
	fmt.Println("f - Quit")
}

/* List of peers known to the server
 * A get request to the url /peers.
 * The server responds with the body containing a list of peer names, one per line.
 */
func getListPeersKnownToServer(client *http.Client) []byte {
	requestUrl := url.URL{Scheme: "https", Host: HOST, Path: "/peers"}
	httpResponseBody, _ := HttpRequest("GET", client, requestUrl.String(), nil, "%s")
	return httpResponseBody
}

/* Peer addresses
 * To locate a peer named p, the client makes a get request to the url /peers/p.
 * The response body contains a json object
 */
func getPeerAddresses(client *http.Client, peersKnownToServer []byte, peerName string) bool {
	requestUrl := url.URL{Scheme: "https", Host: HOST, Path: "/peers"}
	bodyAfterSplit := strings.Split(string(peersKnownToServer), "\n")

	for _, p := range bodyAfterSplit {
		if peerName == p {
			peerUrl := requestUrl.String() + "/" + p
			bodyfromPeer, statusCode := HttpRequest("GET", client, peerUrl, nil, "%s")

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

				return true
			}
		}
	}

	return false
}

/*
 *
 */
func helloToPeerAddress(conn net.PacketConn, peerAddress string, datagramId string, privateKey *ecdsa.PrivateKey) bool {
	for _, peer := range peers {
		for _, address := range peer.Adresses {
			var full_address string
			if peerAddress == fmt.Sprintf("%s:%v", address.Ip, address.Port) {
				if net.ParseIP(address.Ip).To4() == nil {
					full_address = fmt.Sprintf("[%v]:%v", address.Ip, address.Port)
				} else {
					full_address = fmt.Sprintf("%v:%v", address.Ip, address.Port)
				}
				serverAddr, err := net.ResolveUDPAddr("udp", full_address)
				if err != nil {
					log.Fatalf("The method net.ResolveUDPAddr() failed with %s address : %v\n", full_address, err)
				}

				UdpWrite(conn, datagramId, HELLO_TYPE, serverAddr, nil, privateKey)
				return true
			}
		}
	}
	return false
}

/*
 *
 */
func rootRequestToOpenedSession(conn net.PacketConn, peerAddress string, datagramId string, privateKey *ecdsa.PrivateKey) bool {
	for _, session := range sessionsWeOpened {
		if peerAddress == session.FullAddress.String() || peerAddress == fmt.Sprintf("%s:%v", session.FullAddress.IP.String(), session.FullAddress.Port) {
			UdpWrite(conn, datagramId, ROOT_REQUEST_TYPE, session.FullAddress, nil, privateKey)
			return true
		}
	}
	return false
}

/*
 *
 */
func getMerkleTreeAnotherPeer(conn net.PacketConn, peerAddress string, datagramId string, privateKey *ecdsa.PrivateKey) bool {
	for i := 0; i < len(sessionsWeOpened); i++ {
		if peerAddress == sessionsWeOpened[i].FullAddress.String() || peerAddress == fmt.Sprintf("%s:%v", sessionsWeOpened[i].FullAddress.IP.String(), sessionsWeOpened[i].FullAddress.Port) {
			if len(sessionsWeOpened[i].buffer) != 0 {
				writeResult := UdpWrite(conn, datagramId, GET_DATUM_TYPE, sessionsWeOpened[i].FullAddress, sessionsWeOpened[i].buffer[0], privateKey)
				if writeResult {

					getDatumResult := getDatum(conn, i, datagramId, privateKey)
					if getDatumResult || true {
						var messages [][]byte

						// We start from index 1 because the hash of the root is stored in index 0 (we have already used the hash of the root)
						for j := 1; j < len(sessionsWeOpened[i].buffer); j++ {

							if int(sessionsWeOpened[i].buffer[j][HASH_LENGTH+NODE_TYPE_BYTE]) == 0 {
								messages = append(messages, sessionsWeOpened[i].buffer[j][HASH_LENGTH:])
							}
						}

						sessionsWeOpened[i].Merkle = CreateTree(messages, MERKLE_TREE_MAX_ARITY)
						sessionsWeOpened[i].Merkle.DepthFirstSearch(0, ThisPeerMerkleTree.PrintNodesData, nil)
						return true
					}
				}
			}
		}
	}
	return false
}

func getDatum(conn net.PacketConn, sessionIndex int, datagramId string, privateKey *ecdsa.PrivateKey) bool {
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

	if !CheckHash(hash, datagramBody[HASH_LENGTH:]) {
		return false
	}

	for i := 1 + HASH_LENGTH; i < len(datagramBody); i += HASH_LENGTH {
		hashI := datagramBody[i : i+HASH_LENGTH]
		writeResult := UdpWrite(conn, datagramId, GET_DATUM_TYPE, sessionsWeOpened[sessionIndex].FullAddress, hashI, privateKey)
		if writeResult {
			getDatum(conn, sessionIndex, datagramId, privateKey)
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
