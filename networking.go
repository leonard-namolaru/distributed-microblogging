package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"sync"
	"encoding/base64"
	"time"
)

const BUFFER_SIZE = 1500

type WaitingResponse struct {
	FullAddress   *net.UDPAddr // The UDP address from which we are waiting for a reply
	DatagramTypes []int        // A list of the type numbers of the datagrams we are waiting to receive from this address. For example: HELLO_REPLY_TYPE, DATUM_TYPE, NO_DATUM_TYPE
	Id            []byte       // The id that should be in the datagram of the answer we will receive (the same as the id in the datagram of our request)
}

type OpenSession struct {
	FullAddress       *net.UDPAddr
	LastHandshakeTime time.Time
}

type SessionWeOpened struct {
	FullAddress      *net.UDPAddr
	LastDatagramTime time.Time
	Merkle           *MerkleTree
	Buffer           []byte
}

var waitingResponses []WaitingResponse
var openSessions []OpenSession
var sessionsWeOpened []SessionWeOpened
var mutex sync.Mutex

func CreateHttpClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 100
	transport.MaxConnsPerHost = 100
	transport.MaxIdleConnsPerHost = 100
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // This is a code for pedagogical purposes !

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}

	return client
}

func HttpRequest(requestType string, client *http.Client, requestUrl string, data []byte, responseBodyPrintMethod string) ([]byte, int) {
	var req *http.Request
	var errorMessage error
	if DEBUG_MODE {
		fmt.Println()
		log.Printf("HTTP %v REQUEST : %v \n", requestType, requestUrl)

		if requestType == "POST" {
			fmt.Printf("BODY OF THE REQUEST : %s \n", data)
		}
	}

	if requestType == "POST" {
		// func http.NewRequest(method string, url string, body io.Reader) (*http.Request, error)
		req, errorMessage = http.NewRequest(requestType, requestUrl, bytes.NewBuffer(data))
	} else {
		req, errorMessage = http.NewRequest(requestType, requestUrl, nil)
	}

	if errorMessage != nil {
		log.Fatalf("http.NewRequest() function in httpRequest() to %s : %v\n", requestUrl, errorMessage)
	}

	if requestType == "POST" {
		// func (http.Header).Add(key string, value string)
		req.Header.Add("Content-Type", "application/json")
	}

	// func (*http.Client).Do(req *http.Request) (*http.Response, error)
	response, errorMessage := client.Do(req)
	if errorMessage != nil {
		log.Fatalf("client.Do() function in httpRequest() to %s : %v\n", requestUrl, errorMessage)
	}

	// func ioutil.ReadAll(r io.Reader) ([]byte, error)
	responseBody, errorMessage := ioutil.ReadAll(response.Body)
	if errorMessage != nil {
		log.Fatalf("io.ReadAll() function in httpRequest() to %s : %v\n", requestUrl, errorMessage)
	}

	response.Body.Close() // func (io.Closer).Close() error
	if DEBUG_MODE {
		fmt.Printf("HTTP RESPONSE STATUS CODE : %d \n", response.StatusCode)
		fmt.Printf("HTTP RESPONSE BODY :\n"+responseBodyPrintMethod+"\n", responseBody)
	}

	return responseBody, response.StatusCode
}

func UdpRead(conn net.PacketConn, privateKey *ecdsa.PrivateKey, addressesFromServer []Address, publicKeyFromServer *ecdsa.PublicKey) {

	for {
		buf := make([]byte, BUFFER_SIZE)

		_, address, err := conn.ReadFrom(buf)
		if err != nil {
			log.Fatalf("The method conn.ReadFrom() failed in udpRead() : %v \n", err)
		}

		if DEBUG_MODE {
			PrintDatagram(false, address.String(), buf, 0)
		}

		nonSolicitMessage := false
		udpAddress, err := net.ResolveUDPAddr("udp", address.String())
		if err != nil {
			log.Fatalf("The method net.ResolveUDPAddr() failed in udpRead() during the resolve of the address %s : %v \n", address.String(), err)
		}

		addressFind := false
		for _,addr := range addressesFromServer {
			if int(addr.Port) == udpAddress.Port && net.ParseIP(addr.Ip).Equal(udpAddress.IP) && !addressFind {
				if buf[TYPE_BYTE] == 0 || buf[TYPE_BYTE] == 128 || buf[TYPE_BYTE] == byte(ROOT_REQUEST_TYPE) || buf[TYPE_BYTE] == byte(ROOT_TYPE) {
					ok := VerifySignature(buf, publicKeyFromServer )
					if !ok {
						panic(ok)
					}
				}
				addressFind = true
				break
			}
		}

		if !addressFind {
			for _,peer := range peers {
				for _, addr := range peer.Addresses {
					if int(addr.Port) == udpAddress.Port && net.ParseIP(addr.Ip).Equal(udpAddress.IP) && !addressFind {
						if buf[TYPE_BYTE] == 0 || buf[TYPE_BYTE] == 128 || buf[TYPE_BYTE] == byte(ROOT_REQUEST_TYPE) || buf[TYPE_BYTE] == byte(ROOT_TYPE) {
							keyFromPeerBytes, err := base64.RawStdEncoding.DecodeString(peer.Key)
							if err != nil {
								panic(err)
							}
							ok := VerifySignature(buf, ConvertBytesToEcdsaPublicKey(  keyFromPeerBytes ) )
							if !ok {
								panic(ok)
							}

						}
						addressFind = true
						break
					}
				}
			}
		}

		if !addressFind {
			fmt.Println("Response from unknown")
		}

		mutex.Lock()
		i := sliceContainsAddress(waitingResponses, address.String())
		if i != -1 {
			//id := buf[ID_FIRST_BYTE : ID_FIRST_BYTE+ID_LENGTH]
			datagramType := buf[TYPE_BYTE]

			if sliceContainsInt(waitingResponses[i].DatagramTypes, int(datagramType)) != -1 {
				waitingResponses = append(waitingResponses[:i], waitingResponses[i+1:]...)

				// In addition to sessions opened by other peers, we also store sessions we opened
				if buf[TYPE_BYTE] == HELLO_REPLY_TYPE {
					i = sliceContainsSessionWeOpened(sessionsWeOpened, udpAddress.String(), conn, privateKey)
					if i != -1 {
						sessionsWeOpened[i].LastDatagramTime = time.Now()
					} else {
						sessionWeOpened := SessionWeOpened{FullAddress: udpAddress, LastDatagramTime: time.Now(), Merkle: nil, Buffer: nil}
						sessionsWeOpened = append(sessionsWeOpened, sessionWeOpened)
					}
				}

			} else { // We are waiting for a datagram from this address but not a datagram with the received datagram type
				if buf[TYPE_BYTE] >= 128 && buf[TYPE_BYTE] != ERROR_TYPE { // If we receive a response type datagram
					nonSolicitMessage = true
				}
			}
		} else { // If we are not waiting for a message from this peer
			if buf[TYPE_BYTE] >= 128 && buf[TYPE_BYTE] != ERROR_TYPE { // If we receive a response type datagram
				nonSolicitMessage = true
			}
		}
		mutex.Unlock()

		if nonSolicitMessage && buf[TYPE_BYTE] != ERROR_TYPE {
			UdpWrite(conn, string(buf[ID_FIRST_BYTE:ID_FIRST_BYTE+ID_LENGTH]), ERROR_TYPE, udpAddress, []byte("A response type datagram was received even though we did not request such a response"), privateKey)
			continue
		}

		i = sliceContainsSession(openSessions, udpAddress.String())
		if i != -1 {
			if buf[TYPE_BYTE] == HELLO_TYPE {
				openSessions[i].LastHandshakeTime = time.Now()
			}
		} else { // If there is no open session
			if int(buf[TYPE_BYTE]) != HELLO_TYPE && int(buf[TYPE_BYTE]) <= 127 {
				UdpWrite(conn, string(buf[ID_FIRST_BYTE:ID_FIRST_BYTE+ID_LENGTH]), ERROR_TYPE, udpAddress, []byte("No handshake was performed (Hello, HelloReplay) or more than an hour has passed since the last interaction"), privateKey)
				continue
			}
		}

		switch buf[TYPE_BYTE] {
		case byte(HELLO_TYPE): // If a Hello datagram arrives, we send HelloReplay and open a session for an hour
			/*if (buf[FLAGS_LENGTH]>>2) && 1{ //encryption extension

			}*/

			UdpWrite(conn, string(buf[ID_FIRST_BYTE:ID_FIRST_BYTE+ID_LENGTH]), HELLO_REPLY_TYPE, udpAddress, nil, privateKey)
			openSession := &OpenSession{FullAddress: udpAddress, LastHandshakeTime: time.Now()}
			openSessions = append(openSessions, *openSession)
			
			

		case byte(ROOT_REQUEST_TYPE):
			UdpWrite(conn, string(buf[ID_FIRST_BYTE:ID_FIRST_BYTE+ID_LENGTH]), ROOT_TYPE, udpAddress, nil, privateKey)
		case byte(GET_DATUM_TYPE):
			UdpWrite(conn, string(buf[ID_FIRST_BYTE:ID_FIRST_BYTE+ID_LENGTH]), DATUM_TYPE, udpAddress, buf[BODY_FIRST_BYTE:BODY_FIRST_BYTE+GET_DATUM_BODY_LENGTH], privateKey)

		case byte(ROOT_TYPE):
			i = sliceContainsSessionWeOpened(sessionsWeOpened, udpAddress.String(), conn, privateKey)
			rootHash := buf[BODY_FIRST_BYTE : BODY_FIRST_BYTE+ROOT_BODY_LENGTH]
			if i != -1 {
				if sessionsWeOpened[i].Merkle == nil {
					if DEBUG_MODE {
						fmt.Println()
						fmt.Printf("So far we have not created a Merkle tree for this session, so we create a Merkle tree now. \n")
					}
					sessionsWeOpened[i].Merkle = CreateEmptyTree(MERKLE_TREE_MAX_ARITY)
					sessionsWeOpened[i].Buffer = rootHash
				} else {
					if fmt.Sprintf("%x", rootHash) == fmt.Sprintf("%x", sessionsWeOpened[i].Merkle.Root.Hash) {
						if DEBUG_MODE {
							fmt.Println()
							fmt.Printf("The root we got is the same as the root that was stored so far in the Merkle tree for this session. \n")
						}
					} else {
						if DEBUG_MODE {
							fmt.Println()
							fmt.Print("The root we got is not the same as the root that was stored so far in the Merkle tree for this session. We will save the new hash in a buffer until we get the node that this hash represents.\n")
							fmt.Print("If the hash matches the node, we will replace the root of the Merkel tree. \n")
						}
						sessionsWeOpened[i].Buffer = rootHash
					}
				}

			}

		case byte(DATUM_TYPE):
			bodyLength := int(buf[LENGTH_FIRST_BYTE])<<8 | int(buf[LENGTH_FIRST_BYTE+1])

			i = sliceContainsSessionWeOpened(sessionsWeOpened, udpAddress.String(), conn, privateKey)
			if i != -1 {
				sessionsWeOpened[i].Buffer = buf[BODY_FIRST_BYTE : BODY_FIRST_BYTE+bodyLength]
			}

		case byte(NO_DATUM_TYPE):
			bodyLength := int(buf[LENGTH_FIRST_BYTE])<<8 | int(buf[LENGTH_FIRST_BYTE+1])

			i = sliceContainsSessionWeOpened(sessionsWeOpened, udpAddress.String(), conn, privateKey)
			if i != -1 {
				sessionsWeOpened[i].Buffer = buf[BODY_FIRST_BYTE : BODY_FIRST_BYTE+bodyLength]
			}

		}
	}
}

func UdpWrite(conn net.PacketConn, datagramId string, datagramType int, address *net.UDPAddr, data []byte, privateKey *ecdsa.PrivateKey) bool {
	var datagram []byte
	var responseOptions []int
	var responseReceived bool
	var waitingResponse *WaitingResponse
	writingSuccessful := true

	switch datagramType {
	case HELLO_TYPE:
		datagram = HelloOrHelloReplyDatagram(true, datagramId, NAME_FOR_SERVER_REGISTRATION, privateKey)
		responseOptions = append(responseOptions, HELLO_REPLY_TYPE)
	case HELLO_REPLY_TYPE:
		datagram = HelloOrHelloReplyDatagram(false, datagramId, NAME_FOR_SERVER_REGISTRATION, privateKey)
	case ROOT_REQUEST_TYPE:
		datagram = RootRequestDatagram(datagramId, privateKey)
		responseOptions = append(responseOptions, ROOT_TYPE)
	case ROOT_TYPE:
		datagram = RootDatagram(datagramId, privateKey)
	case GET_DATUM_TYPE:
		datagram = GetDatumDatagram(datagramId, data)
		responseOptions = append(responseOptions, NO_DATUM_TYPE, DATUM_TYPE)
	case DATUM_TYPE:
		datagram = DatumDatagram(datagramId, data)
	case ERROR_TYPE:
		datagram = ErrorDatagram(datagramId, data)
	default:
		return false
	}

	waitForResponse := len(responseOptions) != 0
	if waitForResponse {
		mutex.Lock()
		i := sliceContainsAddress(waitingResponses, address.String())
		if i == -1 {
			waitingResponse = &WaitingResponse{FullAddress: address, DatagramTypes: responseOptions}
			waitingResponses = append(waitingResponses, *waitingResponse)
		} else {
			waitingResponses[i].DatagramTypes = append(waitingResponses[i].DatagramTypes, responseOptions...)
		}
		mutex.Unlock()
	}

	responseReceived = false
	for i := 0; !responseReceived && i < 4; i++ {
		// Exponential growth (Croissance exponentielle)
		// Formula : f(x)=a(1+r)^{x}
		// a    =   initial amount
		// r	=	growth rate
		// {x}	=	number of time intervals
		// Source : Google ("Exponential growth Formula")
		r := 1
		timeOut := 0.0
		if waitForResponse {
			timeOut = 2 * math.Pow(float64(1+r), float64(i))
		}

		if DEBUG_MODE {
			PrintDatagram(true, address.String(), datagram, timeOut)
		}

		_, err := conn.WriteTo(datagram, address)
		if err != nil {
			log.Fatalf("The method WriteTo failed in udpWrite() to %s : %v", address.String(), err)
		}

		if waitForResponse {
			time.Sleep(time.Duration(timeOut * float64(time.Second)))
			mutex.Lock()
			if sliceContainsAddress(waitingResponses, address.String()) == -1 {
				responseReceived = true
			} else {
				if i == 3 {
					if DEBUG_MODE {
						log.Printf("AFTER %d ATTEMPTS, WE DID NOT GET THE ANSWER WE EXPECTED FROM %s TO DATAGRAM OF TYPE %d \n", i+1, address.String(), datagramType)
					}
					writingSuccessful = false
				}
			}
			mutex.Unlock()
		} else {
			responseReceived = true
		}
	}

	return writingSuccessful

}

func sliceContainsAddress(slice []WaitingResponse, address string) int {
	for i, element := range slice {
		if element.FullAddress.String() == address {
			return i
		}
	}
	return -1
}

func sliceContainsSession(slice []OpenSession, address string) int {
	for i, element := range slice {
		if element.FullAddress.String() == address {
			// After an hour the session is no longer valid and in that case we remove it
			if time.Since(element.LastHandshakeTime).Minutes() > 55 {
				openSessions = append(openSessions[:i], openSessions[i+1:]...)
				return -1
			} else {
				return i
			}
		}
	}
	return -1
}

func sliceContainsSessionWeOpened(slice []SessionWeOpened, address string, conn net.PacketConn, privateKey *ecdsa.PrivateKey) int {
	for i, element := range slice {
		if element.FullAddress.String() == address {
			return i
		}
	}
	return -1
}

func sliceContainsInt(slice []int, intValue int) int {
	for i, element := range slice {
		if element == intValue {
			return i
		}
	}
	return -1
}
