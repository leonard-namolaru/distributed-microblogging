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

	/* LIST OF PEERS KNOWN TO THE SERVER
	 * A GET REQUEST TO THE URL /peers.
	 * THE SERVER RESPONDS WITH THE BODY CONTAINING A LIST OF PEER NAMES, ONE PER LINE.
	 */
	requestUrl = url.URL{Scheme: "https", Host: HOST, Path: "/peers"}
	httpResponseBody, _ = HttpRequest("GET", httpClient, requestUrl.String(), nil, "%s")

	/* PEER ADDRESSES
	 * TO LOCATE A PEER NAMED p, THE CLIENT MAKES A GET REQUEST TO THE URL /peers/p.
	 * THE RESPONSE BODY CONTAINS A JSON OBJECT
	 */
	var peers []Peer

	bodyAfterSplit := strings.Split(string(httpResponseBody), "\n")
	for _, p := range bodyAfterSplit {

		if len(p) != 0 {
			peerUrl := requestUrl.String() + "/" + p
			bodyfromPeer, statusCode := HttpRequest("GET", httpClient, peerUrl, nil, "%s")

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

	/* HELLO TO (ALL) PEER ADDRESSES
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
					log.Fatalf("The method net.ResolveUDPAddr() failed with %s address : %v\n", full_address, errorMessage)
				}

				UdpWrite(conn, datagram_id, HELLO_TYPE, serverAddr, nil, privateKey)
			}
		}
	}

	/* ROOT REQUEST TO ALL THE SESSIONS WE OPENED
	 */
	for _, session := range sessionsWeOpened {
		// We also have an open session with the server but we are now interested in contacting only the peers with whom we created a session.
		if session.FullAddress.IP.String() != serverUdpAddresses[0].Ip && session.FullAddress.Port != int(serverUdpAddresses[0].Port) || (session.FullAddress.IP.String() != serverUdpAddresses[1].Ip && session.FullAddress.Port != int(serverUdpAddresses[1].Port)) {
			UdpWrite(conn, datagram_id, ROOT_REQUEST_TYPE, session.FullAddress, nil, privateKey)
		}
	}

	/* OBTAINING THE MERKLE TREE FROM ALL THE PEERS WHO GAVE US THE HASH OF THEIR ROOT
	 */
	for i := 0; i < len(sessionsWeOpened); i++ {
		if len(sessionsWeOpened[i].buffer) != 0 {
			writeResult := UdpWrite(conn, datagram_id, GET_DATUM_TYPE, sessionsWeOpened[i].FullAddress, sessionsWeOpened[i].buffer[0], privateKey)
			if writeResult {

				getDatumResult := getDatum(conn, i, datagram_id, privateKey)
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
				}
			}
		}

	}

	fmt.Println()
	fmt.Printf("WAITING FOR NEW MESSAGES ...\n")

	for {
	}

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
