package tcp

import (
	"log"
	"net"
	"sync"
	"time"
)

type TCPServer struct {
	Addr    string
	opt     *tcpOption
	ln      net.TCPListener
	conns   sync.Map // map[net.Conn]struct{}
	wgLn    sync.WaitGroup
	wgConns sync.WaitGroup
}

func NewTCPServer(addr string, opt *tcpOption) *TCPServer {
	if opt == nil {
		log.Fatal("*tcpOption opt can not be nil")
	}
	srv := &TCPServer{
		Addr: addr,
		opt:  opt,
	}
	return srv
}

func (srv *TCPServer) ListenAndServe() error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", srv.Addr)
	if err != nil {
		return err
	}
	ln, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}

	return srv.Serve(ln)
}

func (server *TCPServer) Serve(ln *net.TCPListener) error {
	server.ln = *ln

	server.wgLn.Add(1)
	defer server.wgLn.Done()

	var tempDelay time.Duration
	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Printf("accept error: %v; retrying in %v\n", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		tempDelay = 0

		server.conns.Store(conn, struct{}{})

		//handle tcpConn
		server.wgConns.Add(1)
		go func() {
			defer func() {
				server.conns.Delete(conn)
				server.wgConns.Done()
			}()

			tcpConn := newConn(conn, server.opt)
			defer tcpConn.Close()
			if !server.opt.handler.OnConnect(tcpConn) {
				log.Printf("connect refuse: %v", conn.RemoteAddr().String())
				tcpConn.Close()
				return
			}
			tcpConn.serve(&server.wgConns)
		}()
	}
}

func (server *TCPServer) Close() {
	server.ln.Close()
	server.wgLn.Wait()

	server.conns.Range(
		func(k, v interface{}) bool {
			err := k.(net.Conn).Close()
			if err != nil {
				log.Printf("TCPServer conns close error: %v\n", err)
			}
			return true
		})
	server.wgConns.Wait()
}
