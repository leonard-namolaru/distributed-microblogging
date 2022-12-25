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
)

type WaitingResponse struct {
	FullAddress   net.UDPAddr
	DatagramTypes []int
	Id            []byte
}

type OpenSession struct {
	FullAddress      net.UDPAddr
	LastDatagramTime time.Time
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

func UdpRead(conn net.PacketConn) {
	buf := make([]byte, DATAGRAM_MAX_LENGTH_IN_BYTES)

	for {
		_, address, err := conn.ReadFrom(buf)
		if err != nil {
			log.Fatal("The method conn.ReadFrom() failed in udpRead() : %v \n", err)
		}

		if DEBUG_MODE {
			PrintDatagram(false, address.String(), buf, 0)
		}

		udpAddress, err := net.ResolveUDPAddr("udp", address.String())
		if err != nil {
			log.Fatal("The method net.ResolveUDPAddr() failed in udpRead() during the resolve of the address %s : %v \n", address.String(), err)
		}

		mutex.Lock()
		i := sliceContainsAddress(waitingResponses, address.String())
		if i != -1 {
			//id := buf[ID_FIRST_BYTE : ID_FIRST_BYTE+ID_LENGTH]
			datagramType := buf[TYPE_BYTE]

			if sliceContainsInt(waitingResponses[i].DatagramTypes, int(datagramType)) != -1 {
				waitingResponses = append(waitingResponses[:i], waitingResponses[i+1:]...)
			}
		}
		mutex.Unlock()

		switch buf[TYPE_BYTE] {
		case byte(HELLO_TYPE):
			UdpWrite(conn, string(buf[ID_FIRST_BYTE:ID_FIRST_BYTE+ID_LENGTH]), HELLO_REPLAY_TYPE, udpAddress, nil)
		case byte(ROOT_REQUEST_TYPE):
			UdpWrite(conn, string(buf[ID_FIRST_BYTE:ID_FIRST_BYTE+ID_LENGTH]), ROOT_TYPE, udpAddress, nil)
		case byte(GET_DATUM_TYPE):
			UdpWrite(conn, string(buf[ID_FIRST_BYTE:ID_FIRST_BYTE+ID_LENGTH]), NO_DATUM_TYPE, udpAddress, buf[BODY_FIRST_BYTE:BODY_FIRST_BYTE+GET_DATUM_BODY_LENGTH])
		}
	}
}

func UdpWrite(conn net.PacketConn, datagramId string, datagramType int, address *net.UDPAddr, data []byte) {
	var datagram []byte
	var responseOptions []int
	var responseReceived bool
	var waitingResponse WaitingResponse

	switch datagramType {
	case HELLO_TYPE:
		datagram = HelloDatagram(datagramId, NAME_FOR_SERVER_REGISTRATION)
		responseOptions = append(responseOptions, HELLO_REPLAY_TYPE)
	case HELLO_REPLAY_TYPE:
		datagram = HelloReplayDatagram(datagramId, NAME_FOR_SERVER_REGISTRATION)
	case ROOT_REQUEST_TYPE:
		datagram = RootRequestDatagram(datagramId)
		responseOptions = append(responseOptions, ROOT_TYPE)
	case ROOT_TYPE:
		datagram = RootDatagram(datagramId)
	case GET_DATUM_TYPE:
		datagram = GetDatumDatagram(datagramId, data)
		responseOptions = append(responseOptions, NO_DATUM_TYPE, DATUM_TYPE)
	case NO_DATUM_TYPE:
		datagram = NoDatumDatagram(datagramId, data)
	default:
		return
	}

	waitForResponse := len(responseOptions) != 0
	if waitForResponse {
		mutex.Lock()
		waitingResponse = WaitingResponse{FullAddress: *address, DatagramTypes: responseOptions}
		waitingResponses = append(waitingResponses, waitingResponse)
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
			log.Fatal("The method WriteTo failed in udpWrite() to %s : %v", address.String(), err)
		}

		if waitForResponse {
			time.Sleep(time.Duration(timeOut * float64(time.Second)))
			mutex.Lock()
			if sliceContainsAddress(waitingResponses, waitingResponse.FullAddress.String()) == -1 {
				responseReceived = true
			} else {
				if i == 3 && DEBUG_MODE {
					log.Printf("AFTER %d ATTEMPTS, THERE IS NO ANSWER FROM %s TO DATAGRAM OF TYPE %d \n", i+1, address.String(), datagramType)
				}
			}
			mutex.Unlock()
		} else {
			responseReceived = true
		}
	}

}

func sliceContainsAddress(slice []WaitingResponse, address string) int {
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
