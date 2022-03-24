package tcp

import (
	"log"
	"net"
	"sync"
	"time"
)

type TCPClient struct {
	Addr      string
	ConnNum   int
	opt       *tcpOption
	conns     sync.Map //map[net.Conn]struct{}
	wg        sync.WaitGroup
	closeFlag bool
}

func NewTCPClient(addr string, num int, opt *tcpOption) *TCPClient {
	if err := checkTCPOption(opt); err != nil {
		log.Fatal(err)
	}

	if num <= 0 {
		num = 1
	}

	return &TCPClient{
		Addr:      addr,
		ConnNum:   num,
		opt:       opt,
		conns:     sync.Map{},
		wg:        sync.WaitGroup{},
		closeFlag: false,
	}
}

func (client *TCPClient) Start() {
	for i := 0; i < client.ConnNum; i++ {
		client.wg.Add(1)
		go client.connect()
	}
}

func (client *TCPClient) dial() *net.TCPConn {
	for {
		tcpAddr, err := net.ResolveTCPAddr("tcp", client.Addr)
		if err != nil {
			return nil
		}
		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		if err == nil || client.closeFlag {
			return conn
		}

		log.Printf("connect to %v error: %v\n", client.Addr, err)
		time.Sleep(1 * time.Second)
		continue
	}
}

func (client *TCPClient) connect() {
	defer client.wg.Done()

	conn := client.dial()
	if conn == nil {
		return
	}
	conn.SetKeepAlive(true)

	if client.closeFlag {
		conn.Close()
		return
	}
	client.conns.Store(conn, struct{}{})

	client.wg.Add(1)
	go func() {
		defer func() {
			client.conns.Delete(conn)
			client.wg.Done()
		}()

		tcpConn := newConn(conn, client.opt)
		defer tcpConn.Close()
		if !client.opt.handler.OnConnect(tcpConn) {
			log.Printf("connect refuse: %v\n", conn.RemoteAddr().String())
			tcpConn.Close()
			return
		}
		tcpConn.serve(&client.wg)
	}()

}

func (client *TCPClient) Close() {
	client.closeFlag = true
	client.conns.Range(
		func(k, v interface{}) bool {
			err := k.(*net.TCPConn).Close()
			if err != nil {
				log.Printf("TCPClient conns close error: %v\n", err)
			}
			return true
		})

	client.wg.Wait()
	log.Println("TCPClient close ok")
}
