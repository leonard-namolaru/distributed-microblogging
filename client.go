package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"crypto/ecdsa"
)

type ServerRegistration struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type Peer struct {
	Username string    `json:"name"`
	Addresses []Address `json:"addresses"`
	Key      string    `json:"key"`
}

type Address struct {
	Ip   string `json:"ip"`
	Port uint64 `json:"port"`
}

const DEBUG_MODE = true
const HOST = "jch.irif.fr:8443"
const NAME_FOR_SERVER_REGISTRATION = "HugoLeonardTest3"
const NAME_FILE_PRIVATE_KEY = NAME_FOR_SERVER_REGISTRATION + "_key.priv"
const MERKLE_TREE_MAX_ARITY = 32
const UDP_LISTENING_ADDRESS = ":8083"

var datagramId = "idid"

var peers []Peer
var ThisPeerMerkleTree = CreateTree(CreateMessagesForMerkleTree(33), MERKLE_TREE_MAX_ARITY)

func main() {
	httpClient := CreateHttpClient()

	/* KEY CRYPTOGRAPHY
	 */
	fileInfo, err := os.Stat(NAME_FILE_PRIVATE_KEY)
	myPrivateKey := CreateOrFindPrivateKey(fileInfo, err)
	myPublicKeyEncoded := CreatePublicKeyEncoded(myPrivateKey)

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
	serverRegistration := ServerRegistration{Name: NAME_FOR_SERVER_REGISTRATION, Key: myPublicKeyEncoded}
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
	publicKeyFromServer := ConvertBytesToEcdsaPublicKey(publicKeyFromServerBytes)
	publicKeyFromServerEncoded := base64.RawStdEncoding.EncodeToString(publicKeyFromServerBytes)

	if DEBUG_MODE {
		fmt.Printf("Public Key from server as string : %s\n", publicKeyFromServerEncoded)
	}

	/* HELLO TO EACH OF THE UDP ADDRESSES OF THE SERVER
	 */
	// func net.ListenPacket(network string, address string) (net.PacketConn, error)
	conn, errorMessage := net.ListenPacket("udp", UDP_LISTENING_ADDRESS)
	if errorMessage != nil {
		log.Fatalf("The method net.ListenPacket() failed with %s address : %v\n", UDP_LISTENING_ADDRESS, errorMessage)
	}

	fmt.Println()
	log.Printf("LISTENING TO %s \n", UDP_LISTENING_ADDRESS)

	// The reading of the received datagrams is done in a separate thread

	go UdpRead(conn, myPrivateKey, serverUdpAddresses, publicKeyFromServer)

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

		UdpWrite(conn, datagramId, HELLO_TYPE, serverAddr, nil, myPrivateKey)
		break;
	}

	fmt.Println()
	fmt.Printf("WAITING FOR NEW MESSAGES ...\n")

	var choise string
	var peersKnownToServer []byte = nil

	for {
		fmt.Println()
		printMenu()
		fmt.Println("Your choise : ")
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
			if !helloToPeerAddress(conn, peerAddress, datagramId, myPrivateKey) {
				fmt.Printf("The address %s you specified was not found in the list of addresses of the peers known to the client \n", peerAddress)
			}

		case 'd':
			var peerAddress string
			fmt.Println()
			fmt.Println("ROOT REQUEST TO A OPENED SESSION : ")
			fmt.Println("Enter peer address : ")
			fmt.Scanln(&peerAddress)
			if !rootRequestToOpenedSession(conn, peerAddress, datagramId, myPrivateKey) {
				fmt.Printf("The address %s you specified was not found in the list of addresses of the opened sessions \n", peerAddress)
			}
		case 'e':
			var peerAddress string
			fmt.Println()
			fmt.Println("OBTAIN THE MERKLE TREE FROM ANOTHER PEER WHO GAVE US THE HASH OF ROOT : ")
			fmt.Println("Enter peer address : ")
			fmt.Scanln(&peerAddress)
			if !getMerkleTreeAnotherPeer(conn, peerAddress, datagramId, myPrivateKey) {
				fmt.Printf("The address %s you specified was not found in the list of addresses of the opened sessions or we don't have the hash of the root  \n", peerAddress)
			}
		case 'f':
			ThisPeerMerkleTree.DepthFirstSearch(0, ThisPeerMerkleTree.PrintNodesData, nil)
		case 'g':
			var peerAddress string
			fmt.Println()
			fmt.Println("DISPLAYING ANOTHER PEER'S MERKLE TREE : ")
			fmt.Println("Enter peer address : ")
			fmt.Scanln(&peerAddress)
			if !printMerkleTreeAnotherPeer(conn, peerAddress, datagramId) {
				fmt.Printf("The address %s you specified was not found in the list of addresses of the opened sessions or we don't have a Merkle tree for this session.  \n", peerAddress)
			}
		case 'h':
			var peerAddress string
			fmt.Println()
			fmt.Println("DISPLAYING ANOTHER PEER'S MESSEGES : ")
			fmt.Println("Enter peer address : ")
			fmt.Scanln(&peerAddress)
			if !printLeafFromMerkleTreeAnotherPeer(conn, peerAddress, datagramId) {
				fmt.Printf("The address %s you specified was not found in the list of addresses of the opened sessions or we don't have a Merkle tree for this session.  \n", peerAddress)
			}

		case 'i':
			os.Exit(0)
		default:
			fmt.Println()
			fmt.Printf("Invalid command  \n")
		}
	}

}

func printMenu() {
	str := ""
	str += fmt.Sprintln("----- MENU -----")
	str += fmt.Sprintln("a - List of peers known to the server")
	str += fmt.Sprintln("b - Get peer addresses")
	str += fmt.Sprintln("c - Send Hello to peer address")
	str += fmt.Sprintln("d - Root request to a opened session")
	str += fmt.Sprintln("e - Obtain the merkle tree from another peer who gave us the hash of root")
	str += fmt.Sprintln("f - Print our peer's Merkle tree")
	str += fmt.Sprintln("g - Displaying another peer's Merkle tree")
	str += fmt.Sprintln("h - Displaying another peer's messages")
	str += fmt.Sprintln("i - Quit")
	fmt.Printf(str)
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

					for i, address := range peer.Addresses {
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
		for _, address := range peer.Addresses {
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
			if sessionsWeOpened[i].Buffer != nil && len(sessionsWeOpened[i].Buffer) == HASH_LENGTH {
				writeResult := UdpWrite(conn, datagramId, GET_DATUM_TYPE, sessionsWeOpened[i].FullAddress, sessionsWeOpened[i].Buffer, privateKey)
				if writeResult {
					getDatumResult := getDatum(conn, i, datagramId, privateKey)
					if getDatumResult || true {
						return true
					}
				}
			}
		}
	}
	return false
}

/* A recursive function that obtains the parts that are in the Merkle tree of another peer and are not yet in our possession
 */
func getDatum(conn net.PacketConn, sessionIndex int, datagramId string, privateKey *ecdsa.PrivateKey) bool {
	datagramBody := sessionsWeOpened[sessionIndex].Buffer // We take from the buffer the last node we received

	if len(datagramBody) <= HASH_LENGTH { // Invalid node length
		return false
	}

	hash := datagramBody[0:HASH_LENGTH]
	// If we failed to add the node to the tree (because the hash does not match the content of the node for example)
	if !sessionsWeOpened[sessionIndex].Merkle.AddNode(hash, datagramBody[HASH_LENGTH:]) {
		return false
	}

	// Presentation of the Merkle tree step by step during its construction
	//sessionsWeOpened[sessionIndex].Merkle.DepthFirstSearch(0, sessionsWeOpened[sessionIndex].Merkle.PrintNodesData, nil)

	if datagramBody[HASH_LENGTH+NODE_TYPE_BYTE] == NODE_TYPE_MESSAGE {
		return true
	}

	if datagramBody[HASH_LENGTH+NODE_TYPE_BYTE] == NODE_TYPE_INTERNAL {
		// We go through each of the hashes found in the last node we received
		for i := 1 + HASH_LENGTH; i < len(datagramBody); i += HASH_LENGTH { // 1 for the type byte
			hashI := datagramBody[i : i+HASH_LENGTH]

			// We are looking for the hash in the Merkle tree. If it does not exist the function DepthFirstSearch() returns nil
			if sessionsWeOpened[sessionIndex].Merkle.DepthFirstSearch(0, sessionsWeOpened[sessionIndex].Merkle.GetNodeByHash, hashI) == nil {

				writeResult := UdpWrite(conn, datagramId, GET_DATUM_TYPE, sessionsWeOpened[sessionIndex].FullAddress, hashI, privateKey)
				if writeResult { // The UdpWrite function returns true if we received an answer

					getDatum(conn, sessionIndex, datagramId, privateKey)
				} else {
					return false
				}
			}
		}
	}

	return true
}

/*
 *
 */
func printMerkleTreeAnotherPeer(conn net.PacketConn, peerAddress string, datagramId string) bool {
	for i := 0; i < len(sessionsWeOpened); i++ {
		if peerAddress == sessionsWeOpened[i].FullAddress.String() || peerAddress == fmt.Sprintf("%s:%v", sessionsWeOpened[i].FullAddress.IP.String(), sessionsWeOpened[i].FullAddress.Port) {
			if sessionsWeOpened[i].Merkle != nil {
				sessionsWeOpened[i].Merkle.DepthFirstSearch(0, sessionsWeOpened[i].Merkle.PrintNodesData, nil)
				return true
			}
		}
	}
	return false
}

/*
 *
 */
func printLeafFromMerkleTreeAnotherPeer(conn net.PacketConn, peerAddress string, datagramId string) bool {
	for i := 0; i < len(sessionsWeOpened); i++ {
		if peerAddress == sessionsWeOpened[i].FullAddress.String() || peerAddress == fmt.Sprintf("%s:%v", sessionsWeOpened[i].FullAddress.IP.String(), sessionsWeOpened[i].FullAddress.Port) {
			if sessionsWeOpened[i].Merkle != nil {
				sessionsWeOpened[i].Merkle.DepthFirstSearch(0, sessionsWeOpened[i].Merkle.PrintLeaf, nil)
				return true
			}
		}
	}
	return false
}
