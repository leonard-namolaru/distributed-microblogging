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
	//"math/big"
	//"crypto/rand"
	//"string"
	"bytes"
	"math/rand"
	"bufio"
	"encoding/binary"
	//"strconv"
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

//var p, g, p1, zero, one, a, A, B, s big.Int // a is private key !

var urlAddress, urlRegister, urlPublicKey, urlList url.URL

var myName string
var myId *id

//const NB_BITS = 768 // I don't know if it's the same NB_BITS from the server so I assume NB_BITS = 768 as in the tp
//const NB_BYTES = NB_BITS / 8

var debug bool

func main(){

	debug = true

	rand.Seed(int64(time.Now().Nanosecond())) // initialization rand with nanosecond
	initMyName("Hugo and Lenny")
	initVar()

	me := CreateHttpClient()


	//Step 1 : to get udp address
	body := getHttpResponse(me, urlAddress.String())
	var adressesServer []address
	err := json.Unmarshal(body, &adressesServer)
	if err != nil {
		log.Fatalf("json.Unmarshal() : %v\n", err)
	}

	if debug{
		fmt.Printf("received address : %v:%v and %v:%v\n", adressesServer[0].Ip, adressesServer[0].Port, adressesServer[1].Ip, adressesServer[1].Port)
	}
	

	//Step 2 : to register
	jsonIdentity, err := json.Marshal(myId)
	if err != nil {
		fmt.Println("error, json.Marshal(myId) : %d\n", err)
	}

	if debug{
		fmt.Printf("Convertion id : %v\n", string(jsonIdentity) )
	}

	body = postHttpResponse(me, urlRegister.String(), ([]byte)(jsonIdentity))

	if debug{
		fmt.Printf("identity response : %v\n", body)
	}

	//Step 3 : to get public key of server
	body = getHttpResponse(me, urlPublicKey.String())

	if debug{
		fmt.Printf("public key response : %v", string(body) )
	}
	//the server don't sign their messages recently

	// Step 4: Hello and HelloReply with the same id
	connection, err := net.Dial("udp", fmt.Sprintf("%v:%v", adressesServer[0].Ip, adressesServer[0].Port))
	if err != nil {
		log.Fatalf("net.Dial() : %v\n", err)
	}

	myByteId := CreateRandId()
	myHello := CreateHello(myByteId)

	_, err = connection.Write(myHello)
	if err != nil {
		log.Fatal("Function connection.Write() : ", err)
	}

	err = connection.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err != nil {
		log.Fatal("Function connection.SetReadDeadline() : ", err)
	}

	buffer := make([]byte, 1500)

	_, err = bufio.NewReader(connection).Read(buffer)
	if err != nil {
		log.Fatal("Timeout !")
	}

	/*check := true
	if len(buffer) >= 4 {
		for i := range buffer[:4] {
			if buffer[i] != myByteId[i] {
				check = false
				break
			}
		}
	} else {
		check = false
	}*/

	if debug {
		fmt.Printf("My id (hello) : %v and id received (helloReply) : %v\n", myByteId, buffer[0:4])
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

func postHttpResponse(client *http.Client, url string, data []byte) []byte {

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("io.ReadAll() function : %v", err)
	}
    defer resp.Body.Close()

	return body
}

func postFormHttpResponse(client *http.Client, url string, data url.Values) []byte {

	resp, err := client.PostForm(url, data)
	if err != nil {
		log.Fatalf("client.Do() function : %v\n", err)
	}

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

	/*p.SetString("FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A63A36210000000000090563", 16)
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
	A.Exp(&g, &a, &p)*/

	myId = &id{
		Name: myName,
		Key: "",
	}
}

func initMyName(name string){
	myName = name
}

/*begin := time.Date(2022, 1, 1, 1, 0, 0, 0, time.Local) // first janvier 2022

func CreateMessage(sendedMessage string,receivedHasedMessage string) []byte { //warning hash or no hash ?

	datagramBody := len(sendedMessage)
	datagram := make([]byte, 1+4+32+datagramBody)
	datagram[0] = 0
	duration := time.Since(begin)
	copy(datagram[1:5],duration.MarshalBinary()) // warning : signed or nor signed ?
	copy(datagram,[6:38]byte(receivedHasedMessage))
	datagram[39] = byte(datagram_body_length >> 8)
	datagram[40] = byte(datagram_body_length & 0xFF)
	copy(datagram[1:], []byte(sendedMessage))

	return datagram
}*/

func CreateHello(id []byte) []byte { // signature not implemanted
	datagramLength := 12+len(myId.Name)+1 // if signature are implemanted that's more
	datagramBodyLength := datagramLength-7
	datagram := make([]byte, datagramLength)
	copy(datagram[0:4],id)
	datagram[5] = 0
	datagram[6] = byte(datagramBodyLength >> 8)
	datagram[7] = byte(datagramBodyLength & 0xFF)
	datagram[8] = 0 //recently we don't have implemant extention
	datagram[9] = 0
	datagram[10] = 0
	datagram[11] = 0
	datagram[12] = byte(len(myId.Name))
	copy(datagram[13:13+len(myId.Name)],([]byte)(myId.Name))
	
	return datagram
}

func CreateRandId() []byte {
	id := new(bytes.Buffer)
    err := binary.Write(id, binary.LittleEndian, rand.Int31())
	if err != nil {
		fmt.Println("binary.Write failed in CreateRandId() :", err)
	}
	return id.Bytes()
}