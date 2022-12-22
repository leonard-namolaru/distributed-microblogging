// The source of the comments: the official documentation of GO
package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"net"
	"time"
)

func main() {
	// func net.Dial(network string, address string) (net.Conn, error)
	// Dial connects to the address on the named network.
	// For TCP and UDP networks, the address has the form "host:port".
	connection, errorMessage := net.Dial("udp", "127.0.0.1:8081")
	if errorMessage != nil {
		// func log.Fatal(v ...any)
		// Fatal is equivalent to Print() followed by a call to os.Exit(1).
		log.Fatal("Function net.Dial() : ", errorMessage)
	}

	datagram := []byte{0}
	responseReceived := false
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
			fmt.Printf("%v \n", buffer)
			fmt.Printf("String format : %s \n", buffer)
			responseReceived = true
		}
	}

	// func (net.Conn).Close() error
	// Close closes the connection. Any blocked Read or Write operations will be unblocked and return errors.
	connection.Close()
}

/*
lenny@DESKTOP-DMJ749K:/mnt/c/Users/lenny/OneDrive/Documents/Master 2/Premiere_periode/Protocoles des services Internet$ go run tp6_2021_exercice1_question3.go
We send the following UDP datagramme :
[0]
String format :
Read deadline : 0.200000 sec
We send the following UDP datagramme :
[0]
String format :
Read deadline : 0.400000 sec
We send the following UDP datagramme :
[0]
String format :
Read deadline : 0.800000 sec
We send the following UDP datagramme :
[0]
String format :
Read deadline : 1.600000 sec
We send the following UDP datagramme :
[0]
String format :
Read deadline : 3.200000 sec
2022/11/05 15:49:09 Timeout !
exit status 1
lenny@DESKTOP-DMJ749K:/mnt/c/Users/lenny/OneDrive/Documents/Master 2/Premiere_periode/Protocoles des services Internet$
*/
