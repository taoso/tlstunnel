package main

import (
	"log"
	"net/http"

	proxy "github.com/lvht/tlstunnel/httpproxy"
)

type server struct{}

func main() {
	go func() {
		log.Fatal(http.ListenAndServe("0.0.0.0:8088", proxy.NewRemoteProxy()))
	}()

	log.Fatal(http.ListenAndServe("0.0.0.0:8000", proxy.NewLocalProxy("127.0.0.1:8088")))
}
