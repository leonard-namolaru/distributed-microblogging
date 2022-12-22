package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"time"
)

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

func HttpRequest(requestType string, client *http.Client, requestUrl string, data []byte) []byte {
	var req *http.Request
	var errorMessage error
	if DEBUG_MODE {
		fmt.Println()
		fmt.Printf("HTTP %v REQUEST : %v \n", requestType, requestUrl)

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
		fmt.Printf("HTTP RESPONSE BODY :\n%s \n", responseBody)
	}

	return responseBody
}

func sendUdp2(datagram []byte, address *net.UDPAddr) []byte {
	var buffer []byte

	// func net.Dial(network string, address string) (net.Conn, error)
	_, errorMessage := net.ListenPacket("udp4", "81.194.27.155:1194")
	if errorMessage != nil {
		log.Fatalf("The method net.ListenUDP() failed in sendUdp() to address  %s : %v\n", address, errorMessage)
	}

	return buffer
}

func sendUdp(datagram []byte, address *net.UDPAddr) []byte {
	var buffer []byte

	// func net.Dial(network string, address string) (net.Conn, error)
	connection, errorMessage := net.DialUDP("udp", nil, address)
	if errorMessage != nil {
		log.Fatalf("The method net.DialUDP() failed in sendUdp() to address  %s : %v\n", address, errorMessage)
	}

	responseReceived := false
	for i := 0; !responseReceived; i++ {
		if DEBUG_MODE {
			fmt.Println()
			fmt.Printf("WE SEND A A UDP DATAGRAMME TO : %s \n", address.String())
			fmt.Printf("THE DATAGRAMME AS BYTES : %v \n", datagram)
			fmt.Printf("THE DATAGRAMME AS STRING : %s \n", datagram)
		}

		// func (net.Conn).Write(b []byte) (n int, err error)
		_, errorMessage = connection.Write(datagram)
		if errorMessage != nil {
			log.Fatalf("The method connection.Write() failed in sendUdp() to address  %s : %v\n", address, errorMessage)
		}

		buffer = make([]byte, 1500)

		// Exponential growth (Croissance exponentielle)
		// Formula : f(x)=a(1+r)^{x}
		// a    =   initial amount
		// r	=	growth rate
		// {x}	=	number of time intervals
		// Source : Google ("Exponential growth Formula")
		r := 1
		readDeadline := 2 * math.Pow(float64(1+r), float64(i))

		// func (c *UDPConn) SetReadDeadline(t time.Time) error
		errorMessage = connection.SetReadDeadline(time.Now().Add(time.Duration(readDeadline) * time.Second))
		if errorMessage != nil {
			log.Fatalf("The method connection.SetReadDeadline() failed in sendUdp() to address  %s : %v\n", address, errorMessage)
		}

		if DEBUG_MODE {
			log.Printf("READ DEADLINE : %d SEC \n", int64(readDeadline))
		}

		_, errorMessage = bufio.NewReader(connection).Read(buffer)
		if errorMessage != nil && int64(readDeadline) >= 60 {
			log.Fatalf("Timeout !")
		} else {
			allZero := true
			for i := range buffer {
				if buffer[i] != 0 {
					allZero = false
					break
				}
			}

			if !allZero && DEBUG_MODE {
				responseReceived = true
				//log.Printf("WE RECEIVE THE FOLLOWING UDP DATAGRAMME : \n")
				//log.Printf("THE DATAGRAMME AS BYTES : %v \n", buffer)
				//log.Printf("THE DATAGRAMME AS STRING : %s \n", buffer)
			} else {
				log.Printf("TIMEOUT !")
			}
		}
	}

	connection.Close() // func (net.Conn).Close() error
	return buffer
}
