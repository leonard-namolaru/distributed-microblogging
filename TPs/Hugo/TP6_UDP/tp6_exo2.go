package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	//"strings"
	"time"
	"bufio"
	"math"
)

type jsonUdpAdress struct {
	Host string 	`json:"host"`
	Port uint64 	`json:"port"`
}

const DATAGRAM_MIN_LENGTH = 4 + 1 + 2

const ALPHA = 7/8
const BETA = 3/4

func main() {

	host_url_juliusz := url.URL{Scheme: "https", Host: "jch.irif.fr:8443", Path: "/udp-address.json"}

	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100
	t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := &http.Client{
		Timeout:   10 * time.Second, //si pas de réponse on coupe au bout de 10 sec la connection
		Transport: t,
	}

	jsonbody := getHttpResponse(client, host_url_juliusz.String())
	var udpAdress jsonUdpAdress

	err := json.Unmarshal(jsonbody, &udpAdress)
	if err != nil {
		log.Fatalf("json.Unmarshal() : %v\n", err)
	}

	fmt.Printf("%v:%v\n",udpAdress.Host,udpAdress.Port)

	udpMessage := createUdpMessage("Salut !")

	fmt.Printf("%v \n", udpMessage)
	fmt.Printf("The body : %s \n", udpMessage[7:])
	
	//we create connexion with host_url_udp
	fmt.Printf("%v\n",fmt.Sprintf("%v:%v",udpAdress.Host,udpAdress.Port))
	connection, err := net.Dial("udp", fmt.Sprintf("%v:%v",udpAdress.Host,udpAdress.Port))
	if err != nil {
		log.Fatalf("net.Dial() : %v\n", err)
	}

	reponseReceived := false
	//r := 1
	//var readDeadline float64 = 2

	RTT := 2.0
	RTTvar := 0.0
	RTO := RTT + 4*RTTvar
	beginning := time.Now()

	for RTO <= 60 {

		//we send datagram
		_, err = connection.Write(udpMessage)
		if err != nil {
			log.Fatal("Function connection.Write() : ", err)
		}

		reponse := make([]byte, 1500)

		err = connection.SetReadDeadline(time.Now().Add(time.Duration(RTO) * time.Second))
		if err != nil {
			log.Fatal("Function connection.SetReadDeadline() : ", err)
		}

		readerConnection := bufio.NewReader(connection)
		_, err := readerConnection.Read(reponse)

		if err != nil { // we haven't any reponse
			reponseReceived = false
		} else{
			reponseReceived = true
			fmt.Printf("%v \n", reponse)
			fmt.Printf("The body : %s \n", reponse[7:])
		}
		
		tau := time.Since(beginning).Seconds()
		delta := math.Abs(tau - RTT)
		RTT = ALPHA*RTT + (1.0-ALPHA)*tau
		RTTvar = BETA*RTTvar + (1.0-BETA)*delta
		RTO = RTT + 4*RTTvar

		if reponseReceived {
			time.Sleep(5 * time.Second)
			//if we haven't reponse then we wait the reponse so we don't actualize begginning
			beginning = time.Now()
		}

	}

	connection.Close()
}

func getHttpResponse(client *http.Client, url string) []byte {

	// func http.NewRequest(method string, url string, body io.Reader) (*http.Request, error)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("http.NewRequest() function : %v\n", err)
	}

	// func (*http.Client).Do(req *http.Request) (*http.Response, error)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("client.Do() function : %v\n", err)
	}

	// func io.ReadAll(r io.Reader) ([]byte, error)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("io.ReadAll() function : %v", err)
	}

	defer resp.Body.Close()

	return body
}

func createUdpMessage(message string) []byte {

	datagram_body := []byte(message)
	datagram_body_length := len(datagram_body)
	datagram_length := DATAGRAM_MIN_LENGTH + datagram_body_length

	datagram := make([]byte, datagram_length)

	copy(datagram[:4], []byte("Hugo"))
	datagram[5] = byte(datagram_body_length >> 8)
	datagram[6] = byte(datagram_body_length & 0xFF)

	copy(datagram[7:], datagram_body)

	return datagram
}