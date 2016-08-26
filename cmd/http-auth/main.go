package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/fatedier/frp/src/utils/conn"
)

var (
	Username  string
	Passwd    string
	ProxyAddr string
	BindAddr  string
	BindPort  int
)

func main() {
	flag.StringVar(&Username, "user", "admin", "auth username")
	flag.StringVar(&Passwd, "pwd", "admin", "auth passwd")
	flag.StringVar(&BindAddr, "addr", "0.0.0.0", "bind address")
	flag.IntVar(&BindPort, "p", 9999, "bind port")
	flag.StringVar(&ProxyAddr, "proxy_addr", "127.0.0.1:8080", "redirect proxy address")
	flag.Parse()

	l, err := conn.Listen(BindAddr, int64(BindPort))
	if err != nil {
		fmt.Printf("listen error: %v\n", err)
		os.Exit(1)
	}

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Printf("accept error: %v\n", err)
			os.Exit(1)
		}
		go useAuth(c)
	}
}

func getBadResponse() *http.Response {
	header := make(map[string][]string)
	header["Proxy-Authenticate"] = []string{"Basic"}
	res := &http.Response{
		Status:     "407 Not authorized",
		StatusCode: 407,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     header,
	}
	return res
}

func useAuth(c *conn.Conn) {
	defer c.Close()
	sc, rd := newShareConn(c.TcpConn)
	r, err := http.ReadRequest(bufio.NewReader(rd))
	if err != nil {
		return
	}

	s := strings.SplitN(r.Header.Get("Proxy-Authorization"), " ", 2)
	if len(s) != 2 {
		res := getBadResponse()
		res.Write(c.TcpConn)
		return
	}

	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		res := getBadResponse()
		res.Write(c.TcpConn)
		return
	}

	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		res := getBadResponse()
		res.Write(c.TcpConn)
		return
	}

	if pair[0] != Username || pair[1] != Passwd {
		res := getBadResponse()
		res.Write(c.TcpConn)
		return
	}
	r.Body.Close()

	reDirectConn, err := conn.ConnectServer(ProxyAddr)
	if err != nil {
		return
	}

	var wait sync.WaitGroup
	pipe := func(to net.Conn, from net.Conn) {
		defer to.Close()
		defer from.Close()
		defer wait.Done()

		io.Copy(to, from)
	}

	wait.Add(2)
	go pipe(sc, reDirectConn.TcpConn)
	go pipe(reDirectConn.TcpConn, sc)
	wait.Wait()
	return
}

type sharedConn struct {
	net.Conn
	sync.Mutex
	buff *bytes.Buffer
}

// the bytes you read in io.Reader, will be reserved in sharedConn
func newShareConn(conn net.Conn) (*sharedConn, io.Reader) {
	sc := &sharedConn{
		Conn: conn,
		buff: bytes.NewBuffer(make([]byte, 0, 1024)),
	}
	return sc, io.TeeReader(conn, sc.buff)
}

func (sc *sharedConn) Read(p []byte) (n int, err error) {
	sc.Lock()
	if sc.buff == nil {
		sc.Unlock()
		return sc.Conn.Read(p)
	}
	n, err = sc.buff.Read(p)

	if err == io.EOF {
		sc.buff = nil
		var n2 int
		n2, err = sc.Conn.Read(p[n:])

		n += n2
	}
	sc.Unlock()
	return
}
