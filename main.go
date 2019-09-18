package main

import (
	"encoding/base64"
	"flag"
	"net/http"
	"os"
	"strings"

	"github.com/lvht/tlstunnel/httpproxy"
	"github.com/lvht/tlstunnel/tun"
	"github.com/mholt/certmagic"
	log "github.com/sirupsen/logrus"
)

type server struct{}

var (
	localAddr, remoteAddr, authKey string

	useTLS bool
	useTUN bool
)

func init() {
	os.Setenv("GODEBUG", os.Getenv("GODEBUG")+",tls13=1")

	flag.StringVar(&localAddr, "local", "", "local listen address, 0.0.0.0:80, e.g.")
	flag.StringVar(&remoteAddr, "remote", "", "remote listen address, 0.0.0.0:443, e.g.")
	flag.StringVar(&authKey, "auth", "", "auth key for remote server")
	flag.BoolVar(&useTLS, "tls", false, "enable tls for tunnel(local only)")
	flag.BoolVar(&useTUN, "tun", false, "enable tun for tunnel(local only)")
}

func main() {
	flag.Parse()

	if (localAddr == "" && remoteAddr == "") || authKey == "" {
		flag.Usage()
		os.Exit(1)
	}

	// 使用 Auth-Key 传输，统一做一下 base64 处理
	authKey = base64.StdEncoding.EncodeToString([]byte(authKey))

	var err error
	if localAddr != "" {
		if useTUN {
			err = tun.ClientLoop(authKey, remoteAddr)
		} else {
			handler := httpproxy.NewLocalProxy(remoteAddr, useTLS, nil, authKey)
			err = http.ListenAndServe(localAddr, handler)
		}
	} else {
		handler := httpproxy.NewRemoteProxy(authKey)

		if useTLS {
			certmagic.Default.Email = "mespebapsi@desoz.com"
			certmagic.Default.CA = certmagic.LetsEncryptProductionCA

			ln, err := certmagic.Listen([]string{remoteAddr})
			if err != nil {
				log.Fatal(err)
			}

			err = http.Serve(ln, handler)
		} else {
			remoteAddr = remoteAddr[strings.LastIndex(remoteAddr, ":"):]
			err = http.ListenAndServe(remoteAddr, handler)
		}
	}

	if err != nil {
		log.Fatal(err)
	}
}
