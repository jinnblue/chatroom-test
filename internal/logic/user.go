package logic

import (
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/jinnblue/chatroom-test/pkg/tcp"
)

type User struct {
	IsNew    bool
	EnterAt  time.Time
	UID      int64
	RoomId   uint32
	Nickname string
	Addr     string
	conn     *tcp.TCPConn
}

var globalUID int64 = 0

func NewServerUser(conn *tcp.TCPConn) *User {
	return &User{
		IsNew:    true,
		EnterAt:  time.Now(),
		UID:      atomic.AddInt64(&globalUID, 1),
		Nickname: "",
		Addr:     conn.GetRawConn().RemoteAddr().String(),
		conn:     conn,
	}
}

func NewClientUser(conn *tcp.TCPConn, nickname string) *User {
	return &User{
		IsNew:    true,
		EnterAt:  time.Now(),
		UID:      atomic.AddInt64(&globalUID, 1),
		Nickname: nickname,
		Addr:     conn.GetRawConn().LocalAddr().String(),
		conn:     conn,
	}
}

func (u *User) AsyncSendMessage(msg tcp.Packet) {
	if err := u.conn.AsyncSendPacket(msg); err != nil {
		if !errors.Is(err, tcp.ErrConnClosing) {
			log.Printf("User.AsyncSendMessage user:%v  error:%v\n", u, err)
		}
	}
}

func (u *User) AsyncSendBuff(buf []byte) {
	if err := u.conn.AsyncSendBuff(buf); err != nil {
		if !errors.Is(err, tcp.ErrConnClosing) {
			log.Printf("User.AsyncSendBuff user:%v  error:%v\n", u, err)
		}
	}
}

func (u *User) BuildMessageBuf(msg tcp.Packet) ([]byte, error) {
	return u.conn.BuildMessageBuf(msg)
}

func (u *User) String() string {
	return fmt.Sprintf("UID:%d  Nickname:%s  Addr:%s", u.UID, u.Nickname, u.Addr)
}
