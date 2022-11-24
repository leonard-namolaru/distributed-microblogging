/* Sous Windows, nous pouvons voir quel port le fichier binaire utilise en exécutant le fichier,
 * puis en utilisant la commande netstat -ano. Ensuite, nous arrêtons d'exécuter le fichier
 * et utilisons à nouveau la commande netstat -ano. A ce stade on compare quel port a disparu de la liste,
 * c'est le port utilisé par le fichier binaire !
 */

// The source of the comments: the official documentation of GO
package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
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
	_, errorMessage = bufio.NewReader(connection).Read(buffer)
	if errorMessage != nil {
		log.Fatal("Function bufio.NewReader().Read() : ", errorMessage)
	}

	fmt.Printf("\nWe receive the following UDP datagramme : \n")
	fmt.Printf("%v \n", buffer)
	fmt.Printf("String format : %s \n", buffer)

	// func (net.Conn).Close() error
	// Close closes the connection. Any blocked Read or Write operations will be unblocked and return errors.
	connection.Close()
}

/*
lenny@DESKTOP-DMJ749K:/mnt/c/Users/lenny/OneDrive/Documents/Master 2/Premiere_periode/Protocoles des services Internet$ go run tp6_2021_exercice1_question1.go
We send the following UDP datagramme :
[0]
String format :

We receive the following UDP datagramme :
[89 111 117 32 119 105 108 108 32 102 111 114 103 101 116 32 116 104 97 116 32 121 111 117 32 101 118 101 114 32 107 110 101 119 32 109 101 46]
String format : You will forget that you ever knew me.
lenny@DESKTOP-DMJ749K:/mnt/c/Users/lenny/OneDrive/Documents/Master 2/Premiere_periode/Protocoles des services Internet$
*/

/*
We send the following UDP datagramme :
[0]
String format :

We receive the following UDP datagramme :
[67 104 101 115 115 32 116 111 110 105 103 104 116 46]
String format : Chess tonight.
*/
