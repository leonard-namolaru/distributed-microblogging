// The source of the comments: the official documentation of GO
package main

import (
	"bufio"
	"fmt"
	"log"
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
		errorMessage = connection.SetReadDeadline(time.Now().Add(2 * time.Millisecond))
		if errorMessage != nil {
			log.Fatal("Function connection.SetReadDeadline() : ", errorMessage)
		}

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
lenny@DESKTOP-DMJ749K:/mnt/c/Users/lenny/OneDrive/Documents/Master 2/Premiere_periode/Protocoles des services Internet$ go run tp6_2021_exercice1_question2.go
We send the following UDP datagramme :
[0]
String format :
We send the following UDP datagramme :
[0]
String format :
We send the following UDP datagramme :
[0]
String format :
We send the following UDP datagramme :
[0]
String format :

We receive the following UDP datagramme :
[71 111 111 100 32 100 97 121 32 102 111 114 32 111 118 101 114 99 111 109 105 110 103 32 111 98 115 116 97 99 108 101 115 46 32 32 84 114 121 32 97 32 115 116 101 101 112 108 101 99 104 97 115 101 46]
String format : Good day for overcoming obstacles.  Try a steeplechase.
lenny@DESKTOP-DMJ749K:/mnt/c/Users/lenny/OneDrive/Documents/Master 2/Premiere_periode/Protocoles des services Internet$
*/

/*
lenny@DESKTOP-DMJ749K:/mnt/c/Users/lenny/OneDrive/Documents/Master 2/Premiere_periode/Protocoles des services Internet$ go run tp6_2021_exercice1_question2.go
We send the following UDP datagramme :
[0]
String format :
We send the following UDP datagramme :
[0]
String format :
We send the following UDP datagramme :
[0]
String format :
We send the following UDP datagramme :
[0]
String format :
We send the following UDP datagramme :
[0]
String format :
2022/11/05 14:31:03 Timeout !
exit status 1
lenny@DESKTOP-DMJ749K:/mnt/c/Users/lenny/OneDrive/Documents/Master 2/Premiere_periode/Protocoles des services Internet$
*/
