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
)

type save struct {
	Name string 	`json:"name"`
	Key string		`json:"key"`
}

type clientId struct {
	Username string 			`json:"username"`
	Adresses []adress			`json:"addresses"`
	Key string					`json:"key"`
}

type adress struct {
	Ip string 		`json:"ip"`
	Port uint64 	`json:"port"`
}


func main(){

	urlAddress := url.URL{Scheme: "https", Host: "jch.irif.fr:8443", Path: "/udp-address"}
	//urlRegister := url.URL{Scheme: "https", Host: "jch.irif.fr:8443", Path: "/register"}
	//urlPublicKey := url.URL{Scheme: "https", Host: "jch.irif.fr:8443", Path: "/server-key"}

	me := CreateHttpClient()
	body := getHttpResponse(me, urlAddress.String())

	var adressesServer []adress
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


/*func SaveToServer() bool {

}*/

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

func createClientId(username string, ipv4 string, ipv6 string, port uint64, publicKey string) *clientId {
	adress1Client := &adress{
		Ip: ipv4,
		Port: port,
	}
	adress2Client := &adress{
		Ip: ipv6,
		Port: port,
	}
	adressesClient := []adress{
		*adress1Client,
		*adress2Client,
	}
	client := &clientId{
		Username: username,
		Adresses: adressesClient,
		Key: publicKey,
	}
	return client
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