package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func CreateHttpClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 100
	transport.MaxConnsPerHost = 100
	transport.MaxIdleConnsPerHost = 100
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // This is a code for pedagogical purposes !

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}

	return client
}

func HttpRequest(requestType string, client *http.Client, requestUrl string, data []byte) []byte {
	var req *http.Request
	var errorMessage error
	if DEBUG_MODE {
		fmt.Println()
		fmt.Printf("HTTP %v REQUEST : %v \n", requestType, requestUrl)

		if requestType == "POST" {
			fmt.Printf("BODY OF THE REQUEST : %s \n", data)
		}
	}

	if requestType == "POST" {
		// func http.NewRequest(method string, url string, body io.Reader) (*http.Request, error)
		req, errorMessage = http.NewRequest(requestType, requestUrl, bytes.NewBuffer(data))
	} else {
		req, errorMessage = http.NewRequest(requestType, requestUrl, nil)
	}

	if errorMessage != nil {
		log.Fatalf("http.NewRequest() function in httpRequest() to %s : %v\n", requestUrl, errorMessage)
	}

	if requestType == "POST" {
		// func (http.Header).Add(key string, value string)
		req.Header.Add("Content-Type", "application/json")
	}

	// func (*http.Client).Do(req *http.Request) (*http.Response, error)
	response, errorMessage := client.Do(req)
	if errorMessage != nil {
		log.Fatalf("client.Do() function in httpRequest() to %s : %v\n", requestUrl, errorMessage)
	}

	// func ioutil.ReadAll(r io.Reader) ([]byte, error)
	responseBody, errorMessage := ioutil.ReadAll(response.Body)
	if errorMessage != nil {
		log.Fatalf("io.ReadAll() function in httpRequest() to %s : %v\n", requestUrl, errorMessage)
	}

	response.Body.Close() // func (io.Closer).Close() error
	if DEBUG_MODE {
		fmt.Printf("HTTP RESPONSE BODY :\n%s \n", responseBody)
	}

	return responseBody
}
