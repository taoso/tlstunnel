package main

import (
	"encoding/base64"
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
	localAddr, remoteAddr, authKey string

	useTLS bool
)

func init() {
	os.Setenv("GODEBUG", os.Getenv("GODEBUG")+",tls13=1")

	flag.StringVar(&localAddr, "local", "", "local listen address, 0.0.0.0:80, e.g.")
	flag.StringVar(&remoteAddr, "remote", "", "remote listen address, 0.0.0.0:443, e.g.")
	flag.StringVar(&authKey, "auth", "", "auth key for remote server")
	flag.BoolVar(&useTLS, "tls", false, "enable tls for tunnel(local only)")
}

func main() {
	flag.Parse()

	if (localAddr == "" && remoteAddr == "") || authKey == "" {
		flag.Usage()
		os.Exit(1)
	}

	authKey = base64.StdEncoding.EncodeToString([]byte(authKey))

	var err error
	if localAddr != "" {
		p, err := badhost.NewPool(true)
		if err != nil {
			log.Fatal(err)
		}

		handler := httpproxy.NewLocalProxy(remoteAddr, useTLS, p, authKey)
		err = http.ListenAndServe(localAddr, handler)
	} else {
		handler := httpproxy.NewRemoteProxy(authKey)

		if useTLS {
			certmagic.Default.Email = "mespebapsi@desoz.com"
			certmagic.Default.CA = certmagic.LetsEncryptProductionCA

			err = certmagic.HTTPS([]string{remoteAddr}, handler)
		} else {
			remoteAddr = remoteAddr[strings.LastIndex(remoteAddr, ":"):]
			err = http.ListenAndServe(remoteAddr, handler)
		}
	}

	if err != nil {
		log.Fatal(err)
	}
}
