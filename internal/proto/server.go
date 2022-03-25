package proto

import (
	"github.com/jinnblue/chatroom-test/pkg/tcp"
)

type ServerMsg = tcp.Message

type MsgErrCode int

const (
	UNKNOW MsgErrCode = iota
	LOGIN_OK
	NICK_NAME_EXIST
	ENTER_OK
	INVALID_ROOM_ID
	LEAVE_OK
	NOT_IN_ROOM
)

type SMRespLogin struct {
	ServerMsg
	ErrCode MsgErrCode
}

type SMRespEnter struct {
	ServerMsg
	ErrCode MsgErrCode
}

type SMRespLeave struct {
	ServerMsg
	ErrCode MsgErrCode
}

type SMUserEnter struct {
	ServerMsg
	NickName string
	SendTime int64
}

type SMUserLeave struct {
	ServerMsg
	NickName string
	SendTime int64
}

type SMChatContent struct {
	ServerMsg
	userUID      int64
	NickName     string
	Content      string
	orignContent string
	SendTime     int64
}

func (s *SMChatContent) BackupContent() {
	s.orignContent = s.Content
}

func (s *SMChatContent) GetOrignContent() string {
	return s.orignContent
}

type SMUserStats struct {
	ServerMsg
	NickName string
	Stats    string
}

type SMPopularWord struct {
	ServerMsg
	TheWord string
}
