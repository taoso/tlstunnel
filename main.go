package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/lvht/tlstunnel/httpproxy"
	"github.com/mholt/certmagic"
)

type server struct{}

var (
	localAddr, remoteAddr string

	useTLS bool
)

func init() {
	flag.StringVar(&localAddr, "local", "", "local listen address, 0.0.0.0:80, e.g.")
	flag.StringVar(&remoteAddr, "remote", "", "remote listen address, 0.0.0.0:443, e.g.")
	flag.BoolVar(&useTLS, "tls", false, "enable tls for tunnel(local only)")

	os.Setenv("GODEBUG", os.Getenv("GODEBUG")+",tls13=1")
}

func main() {
	flag.Parse()

	if localAddr == "" && remoteAddr == "" {
		flag.Usage()
		os.Exit(1)
	}

	if localAddr != "" {
		log.Fatal(http.ListenAndServe(localAddr, httpproxy.NewLocalProxy(remoteAddr, useTLS)))
	} else {
		if useTLS {
			certmagic.Default.Email = "mespebapsi@desoz.com"
			certmagic.Default.CA = certmagic.LetsEncryptProductionCA

			log.Fatal(certmagic.HTTPS([]string{remoteAddr}, httpproxy.NewRemoteProxy()))
		} else {
			remoteAddr = remoteAddr[strings.LastIndex(remoteAddr, ":"):]
			log.Fatal(http.ListenAndServe(remoteAddr, httpproxy.NewRemoteProxy()))
		}
	}
}
