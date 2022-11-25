package main

import (
	"fmt"
	"log"
	"net/http"
	"encoding/json"
	"net/url"
	"time"
	"crypto/tls"
	"io"
	"net"
	"math/big"
	"crypto/rand"
	//"string"
)

type id struct {
	Name string 	`json:"name"`
	Key string		`json:"key"`
}

type peerId struct {
	Username string 			`json:"username"`
	Adresses []address			`json:"addresses"`
	Key string					`json:"key"`
}

type address struct {
	Ip string 		`json:"ip"`
	Port uint64 	`json:"port"`
}

var p, g, p1, zero, one, a, A, B, s big.Int // a is private key !

var urlAddress, urlRegister, urlPublicKey, urlList url.URL

var myName string
var myId *id

const NB_BITS = 768 // I don't know if it's the same NB_BITS from the server so I assume NB_BITS = 768 as in the tp
const NB_BYTES = NB_BITS / 8

func main(){

	initMyName("Hugo and Lenny")
	initVar() // B and s are not initialized

	me := CreateHttpClient()
	body := getHttpResponse(me, urlAddress.String())

	var adressesServer []address
	err := json.Unmarshal(body, &adressesServer)
	if err != nil {
		log.Fatalf("json.Unmarshal() : %v\n", err)
	}

	connection, err := net.Dial("udp", fmt.Sprintf("%v:%v", adressesServer[0].Ip, adressesServer[0].Port))
	if err != nil {
		log.Fatalf("net.Dial() : %v\n", err)
	}

	connection.Close()
}

func CreateHttpClient() *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100
	t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := &http.Client{
		Timeout:   10 * time.Second, //if no response break the connection
		Transport: t,
	}

	return client
}

func createPeerId(username string, addressesPeer []address, publicKey string) *peerId {
	peer := &peerId{
		Username: username,
		Adresses: addressesPeer,
		Key: publicKey,
	}
	return peer
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

func postHttpResponse(client *http.Client, url string) []byte {

	// func http.NewRequest(method string, url string, body io.Reader) (*http.Request, error)
	req, err := http.NewRequest("POST", url, nil)
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

func initVar(){

	urlAddress = url.URL{Scheme: "https", Host: "jch.irif.fr:8443", Path: "/udp-address"}
	urlRegister = url.URL{Scheme: "https", Host: "jch.irif.fr:8443", Path: "/register"}
	urlPublicKey = url.URL{Scheme: "https", Host: "jch.irif.fr:8443", Path: "/server-key"}
	urlList = url.URL{Scheme: "https", Host: "jch.irif.fr:8443", Path: "/peers"}

	p.SetString("FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A63A36210000000000090563", 16)
	g.SetInt64(2)
	zero.SetInt64(0)
	one.SetInt64(1)
	p1.Sub(&p, &one)

	buffer_of_bits := make([]byte, NB_BYTES)
	_, errorMessage := rand.Read(buffer_of_bits)
	if errorMessage != nil {
		log.Fatal("rand.Read() function : ", errorMessage)
	}

	a.SetBytes(buffer_of_bits)
	A.Exp(&g, &a, &p)

	myId = &id{
		Name: myName,
		Key: A.String(),
	}
}

func initMyName(name string){
	myName = name
}