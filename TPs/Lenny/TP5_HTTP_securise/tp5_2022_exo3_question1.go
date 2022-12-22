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

	// Check for the presence of an Authorization header
	if req.Header.Get("Authorization") == "" {
		w.Header().Set("www-authenticate", "basic realm=\"tp1\"")

		// func http.Error(w http.ResponseWriter, error string, code int)
		// const http.StatusUnauthorized untyped int = 401
		http.Error(w, "Haha!", http.StatusUnauthorized)
		return
	}

	if req.URL.Path != "/" {
		// func http.NotFound(w http.ResponseWriter, r *http.Request)
		// NotFound replies to the request with an HTTP 404 not found error.
		http.NotFound(w, req)
		return
	}

	w.Header().Set("content-type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<!DOCTYPE html><html><body> Bonjour ! "+req.Header.Get("Authorization")+"</body></html>")
}

const certFile string = "cert.pem"
const keyFile string = "key.pem"

func main() {
	// func http.HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
	// HandleFunc registers the handler function for the given pattern in the DefaultServeMux
	http.HandleFunc("/", handler)

	// func http.ListenAndServeTLS(addr string, certFile string, keyFile string, handler http.Handler) error
	err := http.ListenAndServeTLS(":8080", certFile, keyFile, nil)

	// func log.Fatal(v ...any)
	// Fatal is equivalent to Print() followed by a call to os.Exit(1).
	log.Fatal(err)
}
