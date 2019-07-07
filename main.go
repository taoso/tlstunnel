package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/lvht/tlstunnel/badhost"
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

	var err error
	if localAddr != "" {
		p, err := badhost.NewPool(true)
		if err != nil {
			log.Fatal(err)
		}

		err = http.ListenAndServe(localAddr, httpproxy.NewLocalProxy(remoteAddr, useTLS, p))
	} else {
		if useTLS {
			certmagic.Default.Email = "mespebapsi@desoz.com"
			certmagic.Default.CA = certmagic.LetsEncryptProductionCA

			err = certmagic.HTTPS([]string{remoteAddr}, httpproxy.NewRemoteProxy())
		} else {
			remoteAddr = remoteAddr[strings.LastIndex(remoteAddr, ":"):]
			err = http.ListenAndServe(remoteAddr, httpproxy.NewRemoteProxy())
		}
	}

	if err != nil {
		log.Fatal(err)
	}
}
