package protocol

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"

	"github.com/jinnblue/chatroom-test/pkg/tcp"
)

type GobProtocol struct {
	msgInfo map[string]*MsgInfo
}

type MsgInfo struct {
	msgType    reflect.Type
	msgHandler MsgHandler
}

type MsgHandler func([]interface{})

const NAME_LEN = 2

func NewGobProtocol() *GobProtocol {
	return &GobProtocol{
		msgInfo: make(map[string]*MsgInfo),
	}
}

// Register 注册消息和路由
func (p *GobProtocol) RegisterAndHandle(msg interface{}, h MsgHandler) {
	msgID := p.Register(msg)
	inf, ok := p.msgInfo[msgID]
	if ok {
		inf.msgHandler = h
	}
}

// Register 注册消息
func (p *GobProtocol) Register(msg interface{}) string {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		log.Fatal("gob message pointer required")
	}
	msgID := msgType.Elem().Name()
	if msgID == "" {
		log.Fatal("unnamed gob message")
	}
	if _, ok := p.msgInfo[msgID]; ok {
		log.Fatalf("message %v is already registered", msgID)
	}

	//gob register
	gob.Register(msg)

	inf := new(MsgInfo)
	inf.msgType = msgType
	p.msgInfo[msgID] = inf
	return msgID
}

// SetHandler 设置路由
func (p *GobProtocol) SetHandler(msg interface{}, msgHandler MsgHandler) {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		log.Fatal("gob message pointer required")
	}
	msgID := msgType.Elem().Name()
	i, ok := p.msgInfo[msgID]
	if !ok {
		log.Fatalf("message %v not registered", msgID)
	}

	i.msgHandler = msgHandler
}

// Route 消息路由,goroutine safe
func (p *GobProtocol) Route(msg interface{}, userData interface{}) error {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		return errors.New("gob message pointer required")
	}
	msgID := msgType.Elem().Name()
	i, ok := p.msgInfo[msgID]
	if !ok {
		return fmt.Errorf("message %v not registered", msgID)
	}
	if i.msgHandler != nil {
		i.msgHandler([]interface{}{msg, userData})
	}
	return nil
}

func (p *GobProtocol) GetDecoder(r io.Reader) tcp.ProtDecoder {
	return gob.NewDecoder(r)
}

func (p *GobProtocol) GetEncoder(w io.Writer) tcp.ProtEncoder {
	return gob.NewEncoder(w)
}

// Unmarshal gob反序列化,goroutine safe
func (p *GobProtocol) Unmarshal(data []byte) (interface{}, error) {
	if len(data) < NAME_LEN {
		return nil, errors.New("message too short")
	}
	n := binary.BigEndian.Uint16(data) + NAME_LEN
	if len(data) < int(n) {
		return nil, errors.New("message no name")
	}

	msgID := string(data[NAME_LEN:n])
	inf, ok := p.msgInfo[msgID]
	if !ok {
		return nil, fmt.Errorf("message %v not registered", msgID)
	}

	msg := reflect.New(inf.msgType.Elem()).Interface()
	br := bytes.NewReader(data[n:])
	dec := gob.NewDecoder(br)
	err := dec.Decode(&msg)
	return msg, err
}

// UnmarshalType gob反序列化得到消息类型(MessageType),goroutine safe
func (p *GobProtocol) UnmarshalType(data []byte) (reflect.Type, error) {
	if len(data) < NAME_LEN {
		return nil, errors.New("message too short")
	}
	n := binary.BigEndian.Uint16(data) + NAME_LEN
	if len(data) < int(n) {
		return nil, errors.New("message no name")
	}

	msgID := string(data[NAME_LEN:n])
	inf, ok := p.msgInfo[msgID]
	if !ok {
		return nil, fmt.Errorf("message %v not registered", msgID)
	}
	return inf.msgType.Elem(), nil
}

// Marshal gob序列化,goroutine safe
func (p *GobProtocol) Marshal(msg interface{}) ([]byte, error) {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		return nil, errors.New("gob message pointer required")
	}
	msgID := msgType.Elem().Name()
	if _, ok := p.msgInfo[msgID]; !ok {
		return nil, fmt.Errorf("message %v not registered", msgID)
	}

	n := len(msgID)
	buf := bytes.NewBuffer(make([]byte, 2, 4096))
	// msgid
	binary.BigEndian.PutUint16(buf.Bytes(), uint16(n))
	buf.WriteString(msgID)
	// data
	enc := gob.NewEncoder(buf)
	err := enc.Encode(&msg)
	return buf.Bytes(), err
}
