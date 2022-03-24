package tcp

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"net"
	"reflect"
)

// ------------------
// |  len  |  data	|
// --2byte-----------
type HeaderPacketParser struct {
	Proc Protocol
}

const LEN_BYTES = 2

func NewHeaderPacketParser(prot Protocol) *HeaderPacketParser {
	return &HeaderPacketParser{
		Proc: prot,
	}
}

func (p *HeaderPacketParser) ReadPacket(conn net.Conn) (Packet, error) {
	// read len
	bufMsgLen := make([]byte, LEN_BYTES)
	if _, err := io.ReadFull(conn, bufMsgLen); err != nil {
		return nil, err
	}
	msgLen := binary.BigEndian.Uint16(bufMsgLen)

	// read data
	msgData := make([]byte, msgLen)
	if _, err := io.ReadFull(conn, msgData); err != nil {
		return nil, err
	}

	//unmarshal
	msg, err := p.Proc.Unmarshal(msgData)
	if err != nil {
		return nil, err
	}
	return msg.(Packet), err
}

func (p *HeaderPacketParser) WritePacket(conn net.Conn, msg Packet) (int, error) {
	data, err := p.Proc.Marshal(msg)
	if err != nil {
		return 0, err
	}

	// check len
	msgLen := len(data)
	if msgLen > math.MaxUint16 {
		return 0, errors.New("message too long")
	}

	msgData := make([]byte, LEN_BYTES+msgLen)
	// write len
	binary.BigEndian.PutUint16(msgData, uint16(msgLen))
	// write data
	copy(msgData[LEN_BYTES:], data)

	return conn.Write(msgData)
}

func (p *HeaderPacketParser) BuildPacketBuf(msg Packet) ([]byte, error) {
	data, err := p.Proc.Marshal(msg)
	if err != nil {
		return nil, err
	}

	// check len
	msgLen := len(data)
	if msgLen > math.MaxUint16 {
		return nil, errors.New("message too long")
	}

	msgData := make([]byte, LEN_BYTES+msgLen)
	// write len
	binary.BigEndian.PutUint16(msgData, uint16(msgLen))
	// write data
	copy(msgData[LEN_BYTES:], data)
	return msgData, nil
}

func (p *HeaderPacketParser) ReadBufPacket(inBuf *bufio.Reader) (Packet, error) {
	// read len
	bufMsgLen := make([]byte, LEN_BYTES)
	if _, err := io.ReadFull(inBuf, bufMsgLen); err != nil {
		return nil, err
	}
	msgLen := binary.BigEndian.Uint16(bufMsgLen)

	if inBuf.Buffered() < int(msgLen) {
		// read data
		msgData := make([]byte, msgLen)
		if _, err := io.ReadFull(inBuf, msgData); err != nil {
			return nil, err
		}

		//unmarshal
		msg, err := p.Proc.Unmarshal(msgData)
		if err != nil {
			return nil, err
		}
		return msg.(Packet), err
	}

	//read data msgID len
	buffID, _ := inBuf.Peek(LEN_BYTES)
	msgIDLen := binary.BigEndian.Uint16(buffID)
	n := int(LEN_BYTES + msgIDLen)
	msgData, _ := inBuf.Peek(n)
	//get data msgType
	msgType, err := p.Proc.UnmarshalType(msgData)
	if err != nil {
		inBuf.Discard(int(msgLen))
		return nil, err
	}
	//move to Packet head
	inBuf.Discard(n)

	//decode data to Packet
	msg := reflect.New(msgType).Interface()
	dec := p.Proc.GetDecoder(inBuf)
	err = dec.Decode(&msg)
	return msg.(Packet), err
}

func (p *HeaderPacketParser) WriteBufPacket(outBuf *bufio.Writer, msg Packet) (int, error) {
	data, err := p.Proc.Marshal(msg)
	if err != nil {
		return 0, err
	}

	// check len
	if len(data) > math.MaxUint16 {
		return 0, errors.New("message too long")
	}
	msgLen := uint16(len(data))

	// write len
	if err := outBuf.WriteByte(byte(msgLen >> 8)); err != nil {
		return 0, err
	}
	if err := outBuf.WriteByte(byte(msgLen)); err != nil {
		return 0, err
	}
	// write data
	return outBuf.Write(data)
}
