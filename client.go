package main

import (
	"fmt"
)

type save struct {
	Name string 	`json:"name"`
	Key string		`json:"key"`
}

type adresseClient struct {
	ip string		`json:"ip"`
	port string		`json:"port"`
}

type clientId struct {
	Username string 			`json:"username"`
	Adresses adresseClient		`json:"addresses"`
	key string					`json:"key"`
}


urlAddress := url.URL{Scheme: "https", Host: "jch.irif.fr:8443", Path: "/udp-address"}
urlRegister := url.URL{Scheme: "https", Host: "jch.irif.fr:8443", Path: "/register"}

func CreateHttpClient() *http.client {
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

func createClientId(...) *clientStruct {
	var c clientId
	return c
} 

func Save(client clientId) bool {
	jsonBody, newEtag := postHttpResponse(client, urlRegister.String())
	var response save
	err := json.Unmarshal(jsonBody, &response)
	if err != nil {
		log.Fatalf("json.Unmarshal() : %v\n", err)
	}
	client.Username = response.Name
	client.key = response.key // warning private key or public key
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