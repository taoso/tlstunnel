package tun

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/songgao/water"
)

// ClientLoop 客户端主循环
func ClientLoop(key, server string) (err error) {
	serverName := server

	i := strings.LastIndex(server, ":")
	if i == -1 {
		server = server + ":443"
	}

	c, err := net.Dial("tcp", server)
	if err != nil {
		return
	}
	defer c.Close()

	c = tls.Client(c, &tls.Config{
		ServerName: serverName,
		MinVersion: tls.VersionTLS13,

		ClientSessionCache: tls.NewLRUClientSessionCache(0),
	})

	req := "CONNECT * HTTP/1.1\r\nAuth-Key:" + key + "\r\n\r\n"

	if _, err = c.Write([]byte(req)); err != nil {
		return
	}

	buf := make([]byte, 8)
	if _, err = io.ReadFull(c, buf); err != nil {
		return
	}

	clientIP := net.IP(buf[:4]).String()
	hostIP := net.IP(buf[4:]).String()

	tun, err := water.New(water.Config{DeviceType: water.TUN})
	if err != nil {
		return
	}
	defer tun.Close()

	log.Printf("client %s -> %s", clientIP, hostIP)
	cmd := exec.Command("/sbin/ifconfig", tun.Name(), clientIP, hostIP, "up")
	if err = cmd.Run(); err != nil {
		return
	}

	go func() {
		io.Copy(c, tun)
	}()

	io.Copy(tun, c)
	return
}

// ServerLoop 服务端主循环
func ServerLoop(w http.ResponseWriter, req *http.Request) (err error) {
	hj := w.(http.Hijacker)
	c, _, err := hj.Hijack()
	if err != nil {
		c.Write([]byte("hijack faild"))
		c.Close()
		return
	}
	defer c.Close()

	tun, err := water.New(water.Config{DeviceType: water.TUN})
	if err != nil {
		c.Write([]byte("create tun faild"))
		return
	}
	defer tun.Close()

	hostIP := nextIP()
	clientIP := nextIP()

	defer releaseIP(hostIP)
	defer releaseIP(clientIP)

	log.Printf("host: %s -> %s", hostIP, clientIP)

	if runtime.GOOS != "linux" {
		return errors.New("only support linux for tun server")
	}

	args := []string{tun.Name(), hostIP.String(), "pointopoint", clientIP.String(), "up", "mtu", "65535"}
	if err = exec.Command("/sbin/ifconfig", args...).Run(); err != nil {
		c.Write([]byte("ifconfig faild"))
		return
	}

	if _, err = c.Write(append(clientIP.To4(), hostIP.To4()...)); err != nil {
		return
	}

	go func() {
		io.Copy(c, tun)
	}()

	io.Copy(tun, c)
	return
}
