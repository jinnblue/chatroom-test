package tcp

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// TCPConn Error type
var (
	ErrConnectRefuse = errors.New("connect refuse by server")
	ErrConnClosing   = errors.New("use of closed tcp connection")
	ErrWriteBlocking = errors.New("write packet was blocking")
	ErrReadBlocking  = errors.New("read packet was blocking")
)

const DEFAULT_MAX_FLUSH_GAP = 200 * time.Millisecond

var globalIdx uint32 = 0

type TCPConn struct {
	OnlineIdx      uint32
	opt            *tcpOption
	rawConn        *net.TCPConn
	extraData      interface{}   // to save extra data
	closeFlag      int32         // close flag
	closeChan      chan struct{} // close chan
	packetSendChan chan Packet   // packet send chan
	buffSendChan   chan []byte   // send buff
	packetRecvChan chan Packet   // packet recv chan
	inBuf          *bufio.Reader
	outBuf         *bufio.Writer
}

func newConn(conn *net.TCPConn, opt *tcpOption) *TCPConn {
	return &TCPConn{
		OnlineIdx:      atomic.AddUint32(&globalIdx, 1),
		opt:            opt,
		rawConn:        conn,
		extraData:      nil,
		closeFlag:      0,
		closeChan:      make(chan struct{}),
		packetSendChan: make(chan Packet, opt.sendChanCapLimit),
		buffSendChan:   make(chan []byte, opt.sendChanCapLimit*100),
		packetRecvChan: make(chan Packet, opt.recvChanCapLimit),
		inBuf:          bufio.NewReaderSize(conn, 1024),
		outBuf:         bufio.NewWriterSize(conn, 40960),
	}
}

func (c *TCPConn) GetExtraData() interface{} {
	return c.extraData
}

func (c *TCPConn) SetExtraData(data interface{}) {
	c.extraData = data
}

func (c *TCPConn) GetRawConn() *net.TCPConn {
	return c.rawConn
}

func (c *TCPConn) Close() {
	if atomic.CompareAndSwapInt32(&c.closeFlag, 0, 1) {
		close(c.closeChan)
		close(c.packetSendChan)
		close(c.buffSendChan)
		close(c.packetRecvChan)
		c.inBuf = nil
		c.outBuf = nil
		err := c.rawConn.SetLinger(0)
		if (err != nil) && (!errors.Is(err, net.ErrClosed)) {
			log.Printf("TCPConn.Close() SetLinger err: %v\n", err)
		}
		c.rawConn.Close()
		c.opt.handler.OnClose(c)
	}
}

func (c *TCPConn) IsClosed() bool {
	return atomic.LoadInt32(&c.closeFlag) == 1
}

func (c *TCPConn) BuildMessageBuf(p Packet) (buf []byte, err error) {
	return c.opt.parser.BuildPacketBuf(p)
}

func (c *TCPConn) AsyncSendPacket(p Packet) (err error) {
	if c.IsClosed() {
		return ErrConnClosing
	}

	defer func() {
		if e := recover(); e != nil {
			err = ErrConnClosing
		}
	}()

	select {
	case c.packetSendChan <- p:
		return nil

	default:
		return ErrWriteBlocking
	}
}

func (c *TCPConn) AsyncSendBuff(buf []byte) (err error) {
	if c.IsClosed() {
		return ErrConnClosing
	}

	defer func() {
		if e := recover(); e != nil {
			err = ErrConnClosing
		}
	}()

	select {
	case c.buffSendChan <- buf:
		return nil

	default:
		return ErrWriteBlocking
	}
}

func (c *TCPConn) serve(wg *sync.WaitGroup) {
	wg.Add(2)

	go func() {
		defer wg.Done()
		c.readLoop()
	}()

	go func() {
		defer wg.Done()
		c.writeLoop()
	}()

	c.handleLoop()
}

func (c *TCPConn) readLoop() {
	defer func() {
		// if err := recover(); (err != nil) && (err != io.EOF) {
		// 	log.Printf("TCPConn.readLoop() recover error:%v\n", err)
		// }
		c.Close()
	}()

	for {
		select {
		case <-c.closeChan:
			return

		default:
		}
		if c.IsClosed() {
			return
		}

		// p, err := c.opt.parser.ReadPacket(c.rawConn)
		p, err := c.opt.parser.ReadBufPacket(c.inBuf)
		if err != nil {
			if (err != io.EOF) && (!errors.Is(err, net.ErrClosed)) {
				log.Printf("TCPConn.readLoop() err:%v\n", err)
			}
			return
		}
		if p == nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		if c.IsClosed() {
			return
		}
		c.packetRecvChan <- p
	}
}

func (c *TCPConn) writeLoop() {
	defer func() {
		// if err := recover(); (err != nil) && (err != io.EOF) {
		// 	log.Printf("TCPConn.writeLoop() recover error:%v\n", err)
		// }
		c.Close()
	}()

	var lastFlushTime int64 = 0
	for {
		select {
		case <-c.closeChan:
			return
		case buf := <-c.buffSendChan:
			if c.IsClosed() {
				return
			}

			if _, err := c.outBuf.Write(buf); err != nil {
				if !errors.Is(err, net.ErrClosed) {
					log.Printf("TCPConn.writeLoop() 1 err:%v\n", err)
				}
				return
			}

		case p := <-c.packetSendChan:
			if c.IsClosed() {
				return
			}

			// if _, err := c.opt.parser.WritePacket(c.rawConn, p); err != nil {
			if _, err := c.opt.parser.WriteBufPacket(c.outBuf, p); err != nil {
				if !errors.Is(err, net.ErrClosed) {
					log.Printf("TCPConn.writeLoop() 2 err:%v\n", err)
				}
				return
			}
		}

		now := time.Now().UnixNano()
		if time.Duration(now-lastFlushTime) > DEFAULT_MAX_FLUSH_GAP {
			if err := c.outBuf.Flush(); err != nil {
				if !errors.Is(err, net.ErrClosed) {
					log.Printf("TCPConn.writeLoop() 3 Flush err:%v\n", err)
				}
				return
			}
		}
	}
}

func (c *TCPConn) handleLoop() {
	defer func() {
		// if err := recover(); (err != nil) && (err != io.EOF) {
		// 	log.Printf("TCPConn.handleLoop() recover error:%v\n", err)
		// }
		c.Close()
	}()

	for {
		select {
		case <-c.closeChan:
			return

		case p := <-c.packetRecvChan:
			if !c.opt.handler.OnMessage(c, p) {
				return
			}
		}
	}
}
