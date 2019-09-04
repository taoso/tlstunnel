package httpproxy

import (
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/lvht/tlstunnel/badhost"
	"github.com/lvht/tlstunnel/tun"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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

	tlsCfg := tls.Config{
		ServerName: serverName,
		MinVersion: tls.VersionTLS13,

		ClientSessionCache: tls.NewLRUClientSessionCache(128),
	}

	p.Dial = func(address string) (conn net.Conn, err error) {
		host := address[:strings.LastIndex(address, ":")]
		if pool.HasSuffix(host) {
			goto proxy
		}

		conn, err = net.DialTimeout("tcp", address, 1*time.Second)
		if err == nil {
			return
		}

		log.Infof("remeber bad host: %s, err: %+v", host, err)
		pool.Add(host)

	proxy:
		log.Infof("dial %s via %s", address, remote)
		conn, err = net.Dial("tcp", remote)
		if err != nil {
			err = errors.Wrap(err, "dial to remote faild")
			return
		}

		if useTLS {
			conn = tls.Client(conn, &tlsCfg)
		}

		req := "CONNECT " + address + " HTTP/1.1\r\n" +
			"Auth-Key:" + p.authKey + "\r\n\r\n"
		_, err = conn.Write([]byte(req))
		if err != nil {
			err = errors.Wrap(err, "tunnel handshake faild")
			return
		}

		buf := make([]byte, len(httpConnected))
		_, err = conn.Read(buf[:])
		if err != nil {
			err = errors.Wrap(err, "read handshake status faild")
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
		log.Errorf("dial to upstream err: %+v", err)
		return
	}

	hj := w.(http.Hijacker)
	downConn, _, err := hj.Hijack()
	if err != nil {
		http.Error(w, "cannot hijack", http.StatusInternalServerError)
		return
	}

	defer upConn.Close()
	defer downConn.Close()

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

	timeout := 5 * time.Second
	go func() {
		iocopy(upConn, downConn, timeout)
	}()

	iocopy(downConn, upConn, timeout)
}

func iocopy(dst io.Writer, src io.Reader, timeout time.Duration) {
	size := 32 * 1024
	buf := make([]byte, size)

	timer := time.NewTimer(timeout)
	ch := make(chan bool, 0)

	go func() {
		defer func() { ch <- true }()

		for {
			n, err := src.Read(buf)
			if err != nil {
				log.Debug("read", err)
				return
			}

			n, err = dst.Write(buf[:n])
			if err != nil {
				log.Debug("write", err)
				return
			}

			if !timer.Reset(timeout) {
				return
			}
		}
	}()

	select {
	case <-ch:
		log.Debug("finished")
	case <-timer.C:
		log.Debug("timeout")
	}
	timer.Stop()
}
