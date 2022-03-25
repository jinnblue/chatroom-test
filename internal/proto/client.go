package proto

import (
	"github.com/jinnblue/chatroom-test/pkg/tcp"
)

type ClientMsg = tcp.Message

type CMLogin struct {
	ClientMsg
	NickName string
	SendTime int64
}

type CMEnter struct {
	ClientMsg
	RoomId uint32
}

type CMLeave struct {
	ClientMsg
}

type CMChat struct {
	ClientMsg
	Content  string
	SendTime int64
}

type CommandType int

const (
	POPULAR CommandType = iota
	STATS
)

type CMCommandGM struct {
	ClientMsg
	CmdType CommandType
	Param   string
}
