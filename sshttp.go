package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/oklog/run"
)

func main() {
	var (
		listenAddr = flag.String("listen", ":8022", "listen address")
		sshAddr    = flag.String("ssh", "localhost:22", "SSH proxy address")
		httpAddr   = flag.String("http", "localhost:80", "HTTP proxy address")
	)
	flag.Parse()

	ln, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("listening on %s", *listenAddr)
	log.Printf("proxying HTTP to %s", *httpAddr)
	log.Printf("proxying SSH to %s", *sshAddr)

	var g run.Group
	{
		g.Add(func() error {
			return receive(ln, proxy(*sshAddr), proxy(*httpAddr))
		}, func(error) {
			ln.Close()
		})
	}
	{
		cancel := make(chan struct{})
		g.Add(func() error {
			return interrupt(cancel)
		}, func(error) {
			close(cancel)
		})
	}
	log.Fatal(g.Run())
}

func receive(ln net.Listener, ssh, http func(net.Conn)) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}

		bufconn := newBufferedConn(conn)
		prelude, err := bufconn.Peek()
		if err != nil {
			log.Printf("[%s] Peek: %v", conn.RemoteAddr(), err)
			conn.Close()
			continue
		}
		log.Printf("[%s] Prelude: %s", conn.RemoteAddr(), prelude)

		switch string(prelude) {
		case "SSH":
			go ssh(bufconn)
		default:
			go http(bufconn)
		}
	}
}

func proxy(addr string) func(net.Conn) {
	return func(src net.Conn) {
		log.Printf("[%s] <%s> starting", src.RemoteAddr(), addr)
		defer log.Printf("[%s] <%s> finished", src.RemoteAddr(), addr)

		dst, err := net.Dial("tcp", addr)
		if err != nil {
			log.Printf("[%s] %v", src.RemoteAddr(), err)
			return
		}

		var g run.Group
		g.Add(func() error {
			_, err := io.Copy(dst, src)
			return err
		}, func(error) {
			src.Close()
		})
		g.Add(func() error {
			_, err := io.Copy(src, dst)
			return err
		}, func(error) {
			dst.Close()
		})
		log.Printf("[%s] <%s> %v", src.RemoteAddr(), dst.RemoteAddr(), g.Run())
	}
}

type bufferedConn struct {
	r *bufio.Reader
	net.Conn
}

func newBufferedConn(c net.Conn) bufferedConn {
	return bufferedConn{bufio.NewReaderSize(c, 3), c}
}

func (b bufferedConn) Peek() ([]byte, error) {
	return b.r.Peek(3)
}

func (b bufferedConn) Read(p []byte) (int, error) {
	return b.r.Read(p)
}

func interrupt(cancel <-chan struct{}) error {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-c:
		return fmt.Errorf("received signal %s", sig)
	case <-cancel:
		return nil
	}
}
