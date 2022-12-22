// The source of the comments: the official documentation of GO
package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"
)

const URL = "https://127.0.0.1:8443/udp-address.json"
const DATAGRAM_MIN_LENGTH = 4 + 1 + 2

type Response struct {
	Host string `json:"host"`
	Port int64  `json:"port"`
}

func getHttpResponse(client *http.Client, requestUrl string) []byte {
	fmt.Printf("HTTP GET REQUEST : %v \n", requestUrl)

	// func http.NewRequest(method string, url string, body io.Reader) (*http.Request, error)
	req, errorMessage := http.NewRequest("GET", requestUrl, nil)
	if errorMessage != nil {
		// func log.Fatal(v ...any)
		// Fatal is equivalent to Print() followed by a call to os.Exit(1).
		log.Fatal("http.NewRequest() function : ", errorMessage)
	}

	// func (*http.Client).Do(req *http.Request) (*http.Response, error)
	r, errorMessage := client.Do(req)
	if errorMessage != nil {
		log.Fatal("client.Do() function : ", errorMessage)
	}

	// func ioutil.ReadAll(r io.Reader) ([]byte, error)
	body, errorMessage := ioutil.ReadAll(r.Body)
	// func (io.Closer).Close() error
	r.Body.Close()

	if errorMessage != nil {
		log.Fatal("ioutil.ReadAll() function : ", errorMessage)
	}

	return body
}

func main() {
	transport := &*http.DefaultTransport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // This is a code for pedagogical purposes !
	client := &http.Client{
		Transport: transport,
		Timeout:   50 * time.Second,
	}

	// func getHttpResponse(client *http.Client, requestUrl string) []byte
	body := getHttpResponse(client, URL)
	for _, char := range body {
		fmt.Printf("%v", string(char))
	}

	fmt.Printf("\n")

	var udpAddress Response

	// func json.Unmarshal(data []byte, v any) error
	// Unmarshal parses the JSON-encoded data and stores the result in the value pointed to by v
	errorMessage := json.Unmarshal(body, &udpAddress)
	if errorMessage != nil {
		log.Fatal("json.Unmarshal() function : ", errorMessage)
	}

	fmt.Printf("Host : %s \n", udpAddress.Host)
	fmt.Printf("Port : %d \n\n", udpAddress.Port)

	// func net.Dial(network string, address string) (net.Conn, error)
	// Dial connects to the address on the named network.
	// For TCP and UDP networks, the address has the form "host:port".
	connection, errorMessage := net.Dial("udp", udpAddress.Host+":"+fmt.Sprint(udpAddress.Port))
	if errorMessage != nil {
		// func log.Fatal(v ...any)
		// Fatal is equivalent to Print() followed by a call to os.Exit(1).
		log.Fatal("Function net.Dial() : ", errorMessage)
	}

	datagram_body := []byte("hello")
	datagram_body_length := len(datagram_body)
	datagram_length := DATAGRAM_MIN_LENGTH + datagram_body_length
	var datagram_type byte = 0

	// func make(t Type, size ...int) Type
	// The make built-in function allocates and initializes an object
	datagram := make([]byte, datagram_length)

	// func copy(dst []Type, src []Type) int
	// The copy built-in function copies elements from a source slice into a destination slice.
	// The source and destination may overlap. Copy returns the number of elements copied,
	// which will be the minimum of len(src) and len(dst).

	copy(datagram[:4], []byte("myId")) // The first 4 bytes: for the id
	datagram[4] = datagram_type

	// This expression is a signed right shift
	// Result in the remaining byte value only taking the low 8 bits of the original integer
	// and discarding all the higher bits.
	// Source : https://medium.com/android-news/java-when-to-use-n-8-0xff-and-when-to-use-byte-n-8-2efd82ae7dd7
	datagram[5] = byte(datagram_body_length >> 8)

	// 0xFF is a hexadecimal constant which is 11111111 in binary.
	// By using bitwise AND ( & ) with this constant,
	// it leaves only the last 8 bits of the original
	// Source : https://www.folkstalk.com/tech/0xff-in-python-with-code-examples/
	datagram[6] = byte(datagram_body_length & 0xFF)
	copy(datagram[7:], datagram_body)

	fmt.Printf("We send the following UDP datagramme : \n")
	fmt.Printf("%v \n", datagram)
	fmt.Printf("The body : %s \n", datagram[7:])

	// func (net.Conn).Write(b []byte) (n int, err error)
	// Write writes data to the connection.
	_, errorMessage = connection.Write(datagram)
	if errorMessage != nil {
		log.Fatal("Function connection.Write() : ", errorMessage)
	}

	buffer := make([]byte, 1500)
	_, errorMessage = bufio.NewReader(connection).Read(buffer)
	if errorMessage != nil {
		log.Fatal("Function bufio.NewReader().Read() : ", errorMessage)
	}

	fmt.Printf("\nWe receive the following UDP datagramme : \n")

	check := true
	if len(buffer) >= 4 {
		for i := range buffer[:4] {
			if buffer[i] != datagram[i] {
				check = false
				break
			}
		}
	} else {
		check = false
	}

	if check {
		fmt.Printf("Same id \n")
	} else {
		log.Fatal("Id problem")
	}

	fmt.Printf("%v \n", buffer[:7+int(buffer[6])+int(buffer[7])])
	fmt.Printf("The body : %s \n", buffer[7:])

	// func (net.Conn).Close() error
	// Close closes the connection. Any blocked Read or Write operations will be unblocked and return errors.
	connection.Close()
}
