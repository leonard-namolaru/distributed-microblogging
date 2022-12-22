// The source of the comments: the official documentation of GO
package main

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"time"
)

func httpRequest(client *http.Client, requestUrl string) []byte {
	fmt.Printf("HTTP GET REQUEST : %v \n", requestUrl)

	req, errorMessage := http.NewRequest("GET", requestUrl, nil)
	if errorMessage != nil {
		// Fatal is equivalent to Print() followed by a call to os.Exit(1).
		log.Fatal("http.NewRequest() function : ", errorMessage)
	}

	// func (*http.Client).Do(req *http.Request) (*http.Response, error)
	response, errorMessage := client.Do(req)
	if errorMessage != nil {
		log.Fatal("client.Do() function : ", errorMessage)
	}

	// func ioutil.ReadAll(r io.Reader) ([]byte, error)
	responseBody, errorMessage := ioutil.ReadAll(response.Body)

	if errorMessage != nil {
		log.Fatal("ioutil.ReadAll() function : ", errorMessage)
	}

	// func (io.Closer).Close() error
	response.Body.Close()

	return responseBody
}

const HOST = "127.0.0.1:8081"

func main() {
	// func net.Dial(network string, address string) (net.Conn, error)
	// Dial connects to the address on the named network.
	// For TCP and UDP networks, the address has the form "host:port".
	connection, errorMessage := net.Dial("udp", HOST)
	if errorMessage != nil {
		// func log.Fatal(v ...any)
		// Fatal is equivalent to Print() followed by a call to os.Exit(1).
		log.Fatal("Function net.Dial() : ", errorMessage)
	}

	datagram := []byte{2}
	responseReceived := false
	var response, response_message, response_signature []byte

	for i := 0; i < 5 && !responseReceived; i++ {
		fmt.Printf("We send the following UDP datagramme : \n")
		fmt.Printf("%v \n", datagram)
		fmt.Printf("String format : %s \n", datagram)

		// func (net.Conn).Write(b []byte) (n int, err error)
		// Write writes data to the connection.
		_, errorMessage = connection.Write(datagram)
		if errorMessage != nil {
			log.Fatal("Function connection.Write() : ", errorMessage)
		}

		buffer := make([]byte, 1500)
		// func (c *UDPConn) SetReadDeadline(t time.Time) error
		// SetReadDeadline implements the Conn SetReadDeadline method.

		// Exponential growth (Croissance exponentielle)
		// Formula : f(x)=a(1+r)^{x}
		// a    =   initial amount
		// r	=	growth rate
		// {x}	=	number of time intervals
		// Source : Google ("Exponential growth Formula")
		r := 1
		readDeadline := 200 * math.Pow(float64(1+r), float64(i))
		errorMessage = connection.SetReadDeadline(time.Now().Add(time.Duration(readDeadline) * time.Millisecond)) // 0.2 sec = 200 ms
		if errorMessage != nil {
			log.Fatal("Function connection.SetReadDeadline() : ", errorMessage)
		}
		fmt.Printf("Read deadline : %f sec \n", (readDeadline / 1000))

		_, errorMessage = bufio.NewReader(connection).Read(buffer)
		if errorMessage != nil {
			if i == 4 {
				log.Fatal("Timeout !")
			}
		} else {
			fmt.Printf("\nWe receive the following UDP datagramme : \n")

			for i, element := range buffer {
				if element == 0 {
					response = buffer[0:i]
					break
				}
			}

			for i, element := range response {
				if string(element) == "." {
					response_message = response[:i+1]
					response_signature = response[i+1:]
				}
			}
			responseReceived = true

			fmt.Printf("Response message :  %v \n", response_message)
			fmt.Printf("String format : %s \n", response_message)
			fmt.Printf("Response signature : %v \n", response_signature)

		}
	}

	transport := &*http.DefaultTransport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // This is a code for pedagogical purposes !
	client := &http.Client{
		Transport: transport,
		Timeout:   50 * time.Second,
	}

	publicKeyBytes := httpRequest(client, "https://"+HOST+"/publicKey")
	fmt.Printf("%v \n", publicKeyBytes)

	x, y := elliptic.Unmarshal(elliptic.P256(), publicKeyBytes)
	publicKey := &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}

	hash := sha256.Sum256(response_message)
	//hash := elliptic.Marshal(elliptic.P256(), x, y)
	if ecdsa.VerifyASN1(publicKey, hash[:], response_signature) {
		fmt.Printf("the signatures are identical. \n")
	} else {
		fmt.Printf("the signatures are not identical. \n")
	}

	// func (net.Conn).Close() error
	// Close closes the connection. Any blocked Read or Write operations will be unblocked and return errors.
	connection.Close()
}
