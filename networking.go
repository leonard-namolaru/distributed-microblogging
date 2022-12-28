package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"sync"
	"time"
	"crypto/ecdsa"
)

type WaitingResponse struct {
	FullAddress   *net.UDPAddr
	DatagramTypes []int
	Id            []byte
}

type OpenSession struct {
	FullAddress      *net.UDPAddr
	LastDatagramTime time.Time
}

type SessionWeOpened struct {
	FullAddress      *net.UDPAddr
	LastDatagramTime time.Time
	Merkle           *MerkleTree
	buffer           [][]byte
}

const ERROR_TYPE = 254

const HELLO_TYPE = 0
const HELLO_REPLAY_TYPE = 128

const ROOT_REQUEST_TYPE = 1
const ROOT_TYPE = 129

const GET_DATUM_TYPE = 2
const DATUM_TYPE = 130
const NO_DATUM_TYPE = 131

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

func HttpRequest(requestType string, client *http.Client, requestUrl string, data []byte) ([]byte, int) {
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
		fmt.Printf("HTTP RESPONSE BODY :\n%s \n", responseBody)
	}

	return responseBody, response.StatusCode
}

func UdpRead(conn net.PacketConn, privateKey *ecdsa.PrivateKey) {

	for {
		buf := make([]byte, DATAGRAM_MAX_LENGTH_IN_BYTES+500)

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

		mutex.Lock()
		i := sliceContainsAddress(waitingResponses, address.String())
		if i != -1 {
			//id := buf[ID_FIRST_BYTE : ID_FIRST_BYTE+ID_LENGTH]
			datagramType := buf[TYPE_BYTE]

			if sliceContainsInt(waitingResponses[i].DatagramTypes, int(datagramType)) != -1 {
				waitingResponses = append(waitingResponses[:i], waitingResponses[i+1:]...)

				// In addition to sessions opened by other peers, we also store sessions we opened
				if buf[TYPE_BYTE] == 128 {
					i = sliceContainsSessionWeOpened(sessionsWeOpened, udpAddress.String(), conn, privateKey)
					if i != -1 {
						sessionsWeOpened[i].LastDatagramTime = time.Now()
					} else {
						sessionWeOpened := &SessionWeOpened{FullAddress: udpAddress, LastDatagramTime: time.Now(), Merkle: nil}
						sessionsWeOpened = append(sessionsWeOpened, *sessionWeOpened)
					}
				}

			} else { // We are waiting for a datagram from this address but not a datagram with the received datagram type
				if buf[TYPE_BYTE] >= 128 { // If we receive a response type datagram
					nonSolicitMessage = true
				}
			}
		} else { // If we are not waiting for a message from this peer
			if buf[TYPE_BYTE] >= 128 { // If we receive a response type datagram
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
			if buf[TYPE_BYTE] == 0 {
				openSessions[i].LastDatagramTime = time.Now()
			}
		} else { // If there is no open session
			if int(buf[TYPE_BYTE]) != HELLO_TYPE && int(buf[TYPE_BYTE]) <= 127 {
				UdpWrite(conn, string(buf[ID_FIRST_BYTE:ID_FIRST_BYTE+ID_LENGTH]), ERROR_TYPE, udpAddress, []byte("No handshake was performed (Hello, HelloReplay) or more than an hour has passed since the last interaction"), privateKey)
				continue
			}
		}

		switch buf[TYPE_BYTE] {
		case byte(HELLO_TYPE): // If a Hello datagram arrives, we send HelloReplay and open a session for an hour
			UdpWrite(conn, string(buf[ID_FIRST_BYTE:ID_FIRST_BYTE+ID_LENGTH]), HELLO_REPLAY_TYPE, udpAddress, nil, privateKey)
			openSession := &OpenSession{FullAddress: udpAddress, LastDatagramTime: time.Now()}
			openSessions = append(openSessions, *openSession)

		case byte(ROOT_REQUEST_TYPE):
			UdpWrite(conn, string(buf[ID_FIRST_BYTE:ID_FIRST_BYTE+ID_LENGTH]), ROOT_TYPE, udpAddress, nil, privateKey)
		case byte(GET_DATUM_TYPE):
			UdpWrite(conn, string(buf[ID_FIRST_BYTE:ID_FIRST_BYTE+ID_LENGTH]), DATUM_TYPE, udpAddress, buf[BODY_FIRST_BYTE:BODY_FIRST_BYTE+GET_DATUM_BODY_LENGTH], privateKey)

		case byte(ROOT_TYPE):
			i = sliceContainsSessionWeOpened(sessionsWeOpened, udpAddress.String(), conn, privateKey)
			if i != -1 {
				sessionsWeOpened[i].buffer = append(sessionsWeOpened[i].buffer, buf[BODY_FIRST_BYTE:BODY_FIRST_BYTE+ROOT_BODY_LENGTH])
			}

		case byte(DATUM_TYPE):
			bodyLength := int(buf[LENGTH_FIRST_BYTE]) + int(buf[LENGTH_FIRST_BYTE+1])

			i = sliceContainsSessionWeOpened(sessionsWeOpened, udpAddress.String(), conn, privateKey)
			if i != -1 {
				sessionsWeOpened[i].buffer = append(sessionsWeOpened[i].buffer, buf[BODY_FIRST_BYTE:BODY_FIRST_BYTE+bodyLength])
			}

		case byte(NO_DATUM_TYPE):
			bodyLength := int(buf[LENGTH_FIRST_BYTE]) + int(buf[LENGTH_FIRST_BYTE+1])

			i = sliceContainsSessionWeOpened(sessionsWeOpened, udpAddress.String(), conn, privateKey)
			if i != -1 {
				sessionsWeOpened[i].buffer = append(sessionsWeOpened[i].buffer, buf[BODY_FIRST_BYTE:BODY_FIRST_BYTE+bodyLength])
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
		datagram = HelloDatagram(datagramId, NAME_FOR_SERVER_REGISTRATION, privateKey)
		responseOptions = append(responseOptions, HELLO_REPLAY_TYPE)
	case HELLO_REPLAY_TYPE:
		datagram = HelloReplayDatagram(datagramId, NAME_FOR_SERVER_REGISTRATION, privateKey)
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
			if sliceContainsAddress(waitingResponses, waitingResponse.FullAddress.String()) == -1 {
				responseReceived = true
			} else {
				if i == 3 && DEBUG_MODE {
					log.Printf("AFTER %d ATTEMPTS, THERE IS NO ANSWER FROM %s TO DATAGRAM OF TYPE %d \n", i+1, address.String(), datagramType)
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
			if time.Since(element.LastDatagramTime).Minutes() > 55 {
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
			// After an hour the session is no longer valid
			if time.Since(element.LastDatagramTime).Minutes() > 55 {
				// We resend a Hello message, if we receive HelloReplay as a response
				// (the write function will return true), we succeed in renewing the session
				if UdpWrite(conn, string([]byte{0, 0, 0, 0}), HELLO_TYPE, element.FullAddress, nil, privateKey) {
					return i
				} else {
					sessionsWeOpened = append(sessionsWeOpened[:i], sessionsWeOpened[i+1:]...) // We remove the session
					return -1
				}
			} else {
				return i
			}
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
