package main

import (
	"net/http"
)

var serveMux = http.NewServeMux()

var httpServer = http.Server{
	Addr:    ":8080",
	Handler: serveMux,
}

func main() {
	serveMux.Handle("/", http.FileServer(http.Dir("./")))

	httpServer.ListenAndServe()

}
