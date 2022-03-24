package tcp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
)

type ProtDecoder interface {
	Decode(e interface{}) error
}

type ProtEncoder interface {
	Encode(e interface{}) error
}

// must goroutine safe
type Protocol interface {
	Route(msg interface{}, userData interface{}) error
	Unmarshal(data []byte) (interface{}, error)
	UnmarshalType(data []byte) (reflect.Type, error)
	Marshal(msg interface{}) ([]byte, error)
	GetDecoder(r io.Reader) ProtDecoder
	GetEncoder(w io.Writer) ProtEncoder
}

type Packet interface {
	fmt.Stringer
	// String() string
}

type PacketParser interface {
	ReadPacket(conn net.Conn) (Packet, error)
	WritePacket(conn net.Conn, msg Packet) (int, error)
	ReadBufPacket(inBuf *bufio.Reader) (Packet, error)
	WriteBufPacket(outBuf *bufio.Writer, msg Packet) (int, error)
	BuildPacketBuf(msg Packet) ([]byte, error)
}

type Handler interface {
	OnConnect(*TCPConn) bool
	OnMessage(*TCPConn, Packet) bool
	OnClose(*TCPConn)
}

type MessageType uint8

const (
	SERVER_TO_CLIENT MessageType = iota
	CLIENT_TO_SERVER
)

type Message struct{}

func (m *Message) String() string {
	bytes, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("Message String() err:%w", err).Error()
	}
	return string(bytes)
}

const (
	DEFAULT_SEND_CHAN_LIMIT = 8
	DEFAULT_RECV_CHAN_LIMIT = 32
)

type tcpOption struct {
	handler          Handler
	parser           PacketParser
	sendChanCapLimit int
	recvChanCapLimit int
}

type TCPOptionFn func(opt *tcpOption)

func NewTCPOption(h Handler, p PacketParser, opts ...TCPOptionFn) *tcpOption {
	if h == nil {
		log.Fatalln("Handler h can not be nil")
	}
	if p == nil {
		log.Fatalln("PacketParser p can not be nil")
	}

	option := &tcpOption{
		handler: h,
		parser:  p,
	}

	//opt func set
	for _, opt := range opts {
		opt(option)
	}

	if option.sendChanCapLimit <= 0 {
		option.sendChanCapLimit = DEFAULT_SEND_CHAN_LIMIT
	}
	if option.sendChanCapLimit <= 0 {
		option.recvChanCapLimit = DEFAULT_RECV_CHAN_LIMIT
	}

	return option
}

func WithSendChanLimit(limit int) TCPOptionFn {
	return func(opt *tcpOption) {
		opt.sendChanCapLimit = limit
	}
}

func WithRecvChanLimit(limit int) TCPOptionFn {
	return func(opt *tcpOption) {
		opt.recvChanCapLimit = limit
	}
}

var (
	ErrTCPOptionNil = errors.New("tcpOption can not be nil")
	ErrHandlerIsNil = errors.New("tcpOption.handler can not be nil")
	ErrParserIsNil  = errors.New("tcpOption.parser can not be nil")
)

func checkTCPOption(opt *tcpOption) error {
	if opt == nil {
		return ErrTCPOptionNil
	}
	if opt.handler == nil {
		return ErrHandlerIsNil
	}
	if opt.parser == nil {
		return ErrParserIsNil
	}

	return nil
}
