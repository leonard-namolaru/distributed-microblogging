// The source of the comments: the official documentation of GO

// A package clause begins each source file and defines the package to which the file belongs.
// A set of files sharing the same PackageName form the implementation of a package
// An implementation may require that all source files for a package inhabit the same directory.
package main

// An import declaration states that the source file containing the declaration depends on functionality of the imported package
import (
	// Package bytes implements functions for the manipulation of byte slices.
	"bytes"

	// Package crypto collects common cryptographic constants.
	// Package tls partially implements TLS 1.2, as specified in RFC 5246, and TLS 1.3, as specified in RFC 8446.
	"crypto/tls"

	// Package fmt implements formatted I/O with functions analogous to C's printf and scanf.
	"fmt"

	// Package io provides basic interfaces to I/O primitives.
	// Package ioutil implements some I/O utility functions.
	"io/ioutil"

	// Package log implements a simple logging package.
	"log"

	// Package net provides a portable interface for network I/O,
	// including TCP/IP, UDP, domain name resolution, and Unix domain sockets.
	// Package http provides HTTP client and server implementations.
	"net/http"

	// Package time provides functionality for measuring and displaying time.
	"time"

	// Package os provides a platform-independent interface to operating system functionality.
	"os"
)

const CHAT_URL = "https://jch.irif.fr:8082/chat/"
const EXIT_FAILURE = 1
const MESSEGES_NUM = 50

func getHttpResponse(client *http.Client, requestUrl string) []byte {
	fmt.Printf("HTTP GET REQUEST : %v \n", requestUrl)

	// func http.NewRequest(method string, url string, body io.Reader) (*http.Request, error)
	req, errorMessage := http.NewRequest("GET", requestUrl, nil)
	if errorMessage != nil {
		log.Printf("http.NewRequest() function : %v", errorMessage)
		// func os.Exit(code int)
		os.Exit(EXIT_FAILURE)
	}

	// func (*http.Client).Do(req *http.Request) (*http.Response, error)
	r, errorMessage := client.Do(req)
	if errorMessage != nil {
		log.Printf("client.Do() function : %v", errorMessage)
		// func os.Exit(code int)
		os.Exit(EXIT_FAILURE)
	}

	// func ioutil.ReadAll(r io.Reader) ([]byte, error)
	body, errorMessage := ioutil.ReadAll(r.Body)
	// func (io.Closer).Close() error
	r.Body.Close()

	if errorMessage != nil {
		// func log.Printf(format string, v ...any)
		log.Printf("ioutil.ReadAll() function : %v", errorMessage)
		// func os.Exit(code int)
		os.Exit(EXIT_FAILURE)
	}

	return body
}

// A complete program is created by linking a single, unimported package called the main package with all the packages it imports, transitively.
// The main package must have package name main and declare a function main that takes no arguments and returns no value.
// Program execution begins by initializing the main package and then invoking the function main
func main() {
	transport := &*http.DefaultTransport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // This is a code for pedagogical purposes !
	client := &http.Client{
		Transport: transport,
		Timeout:   50 * time.Second,
	}

	// func getHttpResponse(client *http.Client, requestUrl string) []byte
	body := getHttpResponse(client, CHAT_URL)
	for _, char := range body {
		fmt.Printf("%v", string(char))
	}

	fmt.Printf("\n")

	// For exemple :
	// If there is only 1 message in the chat :
	// body == [f, 7, 6, e, a, 3, d, 8, c, 2, d, 8, 6, 2, d, f, a, 6, 3, d, 4, 3, 4, b, f, 5, 0, f, 4, 0, 5, 6, '\n']

	// func bytes.Split(s []byte, sep []byte) [][]byte
	// Split slices s into all subslices separated by sep and
	// returns a slice of the subslices between those separators.
	// If sep is empty, Split splits after each UTF-8 sequence.
	ids := bytes.Split(body, []byte{byte('\n')})

	// For exemple :
	// If there are 2 messages in the chat :
	// ids == ["f76ea3d8c2d862dfa63d434bf50f4056", "23f55b4b93b2ac205f84f6ad708de6bb", ""]

	if len(ids) > 0 {
		// func len(v Type) int
		last := len(ids) - 1

		if len(ids[last]) == 0 {
			// Slice expressions construct a substring or slice from a string, array, pointer to array, or slice.
			// There are two variants: a simple form that specifies a low and high bound,
			// and a full form that also specifies a bound on the capacity.
			ids = ids[:last]
		}
	}

	fmt.Printf("Messeges in the chat : %d \n", len(ids))
	fmt.Printf("%d last messages : \n\n", MESSEGES_NUM)

	forBeginning := 0
	if len(ids) > MESSEGES_NUM {
		forBeginning = len(ids) - MESSEGES_NUM
	}

	for i := forBeginning; i < len(ids); i++ {
		// func fmt.Printf(format string, a ...any) (n int, err error)
		fmt.Printf("Id %v: %v\n", i, string(ids[i]))

		httpGetMessage := CHAT_URL + string(ids[i])
		body = getHttpResponse(client, httpGetMessage)

		fmt.Printf("Message: ")
		for _, char := range body {
			fmt.Printf("%v", string(char))
		}
		fmt.Printf("\n\n")

	}
}
