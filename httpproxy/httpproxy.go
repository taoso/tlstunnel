package httpproxy

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"
)

// Proxy http 隧道服务
type Proxy struct {
	Dial func(address string) (net.Conn, error)
}

var httpConnected = []byte("HTTP/1.1 200 Connection established\r\n\r\n")

// NewLocalProxy 本地隧道服务
// 通过 http CONNECT 请求要求远程服务建立隧道
func NewLocalProxy(remote string, useTLS bool) *Proxy {
	serverName := remote

	i := strings.LastIndex(remote, ":")
	if i == -1 {
		remote = remote + ":443"
	} else {
		serverName = remote[:i]
	}

	p := &Proxy{}

	p.Dial = func(address string) (conn net.Conn, err error) {
		conn, err = net.DialTimeout("tcp", remote, 500*time.Millisecond)
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

		_, err = conn.Write([]byte("CONNECT " + address + " HTTP/1.1\r\n\r\n"))
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
func NewRemoteProxy() *Proxy {
	return &Proxy{
		Dial: func(address string) (net.Conn, error) {
			return net.DialTimeout("tcp", address, 500*time.Millisecond)
		},
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var host string
	if req.Method == http.MethodConnect {
		host = req.RequestURI
		if strings.LastIndex(host, ":") == -1 {
			host += ":443"
		}
	} else {
		host = req.Host
		if strings.LastIndex(host, ":") == -1 {
			host += ":80"
		}
	}

	upConn, err := p.Dial(host)
	if err != nil {
		http.Error(w, "cannot connect to upstream", http.StatusBadGateway)
		return
	}

	hj := w.(http.Hijacker)
	downConn, _, err := hj.Hijack()
	if err != nil {
		http.Error(w, "cannot hijack", http.StatusInternalServerError)
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
