// The source of the comments: the official documentation of GO
package main

import (
	// Package fmt implements formatted I/O with functions analogous to C's printf and scanf
	"fmt"

	// Package fmt implements formatted I/O with functions analogous to C's printf and scanf
	"log"

	// Package http provides HTTP client and server implementations.
	"net/http"
)

func handler(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		// func http.NotFound(w http.ResponseWriter, r *http.Request)
		// NotFound replies to the request with an HTTP 404 not found error.
		http.NotFound(w, req)
		return
	}

	w.Header().Set("content-type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<!DOCTYPE html><html><body> Bonjour ! </body></html>")
}

const certFile string = "cert.pem"
const keyFile string = "key.pem"

func main() {
	// func http.HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
	// HandleFunc registers the handler function for the given pattern in the DefaultServeMux
	http.HandleFunc("/", handler)

	// func http.ListenAndServe(addr string, handler http.Handler) error
	// ListenAndServe listens on the TCP network address addr
	// and then calls Serve with handler to handle requests on incoming connections.
	// Accepted connections are configured to enable TCP keep-alives.
	// The handler is typically nil, in which case the DefaultServeMux is used.
	// ListenAndServe always returns a non-nil error.
	// err := http.ListenAndServe(":8080", nil)

	// func http.ListenAndServeTLS(addr string, certFile string, keyFile string, handler http.Handler) error
	// ListenAndServeTLS acts identically to ListenAndServe, except that it expects HTTPS connections.
	// Additionally, files containing a certificate and matching private key for the server must be provided.
	// If the certificate is signed by a certificate authority, the certFile should be the concatenation
	// of the server's certificate, any intermediates, and the CA's certificate.
	err := http.ListenAndServeTLS(":8080", certFile, keyFile, nil)

	// func log.Fatal(v ...any)
	// Fatal is equivalent to Print() followed by a call to os.Exit(1).
	log.Fatal(err)
}
