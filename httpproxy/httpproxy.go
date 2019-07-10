package httpproxy

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/lvht/tlstunnel/badhost"
	"github.com/lvht/tlstunnel/tun"
)

// Proxy http 隧道服务
type Proxy struct {
	Dial func(address string) (net.Conn, error)

	authKey string

	isRemote bool

	serverName string
}

var slogan = []byte("Across the Great Wall we can reach every corner in the world.")

var httpConnected = []byte("HTTP/1.1 200 Connection established\r\n\r\n")

// NewLocalProxy 本地隧道服务
// 通过 http CONNECT 请求要求远程服务建立隧道
func NewLocalProxy(remote string, useTLS bool, pool *badhost.Pool, authKey string) *Proxy {
	serverName := remote

	i := strings.LastIndex(remote, ":")
	if i == -1 {
		remote = remote + ":443"
	} else {
		serverName = remote[:i]
	}

	p := &Proxy{authKey: authKey, serverName: serverName}

	p.Dial = func(address string) (conn net.Conn, err error) {
		host := address[:strings.LastIndex(address, ":")]
		if pool.HasSuffix(host) {
			goto proxy
		}

		conn, err = net.DialTimeout("tcp", address, 300*time.Millisecond)
		if err == nil {
			log.Printf("dial %s", address)
			return
		}

		log.Printf("remeber bad host: %s, err: %+v", host, err)
		pool.Add(host)

	proxy:
		log.Printf("dial %s via %s", address, remote)
		conn, err = net.DialTimeout("tcp", remote, 1*time.Second)
		if err != nil {
			return
		}

		if useTLS {
			conn = tls.Client(conn, &tls.Config{
				ServerName: serverName,
				MinVersion: tls.VersionTLS13,

				ClientSessionCache: tls.NewLRUClientSessionCache(0),
			})
		}

		req := "CONNECT " + address + " HTTP/1.1\r\n" +
			"Auth-Key:" + p.authKey + "\r\n\r\n"
		_, err = conn.Write([]byte(req))
		if err != nil {
			return
		}

		buf := make([]byte, len(httpConnected))
		_, err = conn.Read(buf[:])
		if err != nil {
			return
		}

		if !bytes.Equal(buf, httpConnected) {
			err = errors.New(string(buf))
		}

		return
	}

	return p
}

// NewRemoteProxy 远各隧道服务
// 发送客户端请求到目标服务器
func NewRemoteProxy(authKey string) *Proxy {
	return &Proxy{
		Dial: func(address string) (net.Conn, error) {
			return net.DialTimeout("tcp", address, 500*time.Millisecond)
		},
		isRemote: true,
		authKey:  authKey,
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var host string
	if req.Method == http.MethodConnect {
		if p.isRemote && req.Header.Get("Auth-Key") != p.authKey {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		host = req.RequestURI

		// CONNECT * HTTP/1.1 请求表示建立 tun 隧道
		if host == "*" {
			tun.ServerLoop(w, req)
			return
		}

		if strings.LastIndex(host, ":") == -1 {
			host += ":443"
		}
	} else {
		host = req.Host
		if strings.LastIndex(host, ":") == -1 {
			host += ":80"
		}

		if p.isRemote && host == p.serverName {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "text/plain")
			w.Write(slogan)
			return
		}
	}

	upConn, err := p.Dial(host)
	if err != nil {
		http.Error(w, "cannot connect to upstream", http.StatusBadGateway)
		log.Println("dial to upstream err: ", err)
		return
	}

	hj := w.(http.Hijacker)
	downConn, _, err := hj.Hijack()
	if err != nil {
		http.Error(w, "cannot hijack", http.StatusInternalServerError)
		downConn.Close()
		return
	}

	if req.Method == http.MethodConnect {
		downConn.Write(httpConnected)
	} else {
		dump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			http.Error(w, "cannot dump rquest", http.StatusInternalServerError)
			return
		}
		upConn.Write(dump)
	}

	go func() {
		io.Copy(upConn, downConn)
	}()

	io.Copy(downConn, upConn)
}
