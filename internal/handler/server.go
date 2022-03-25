package handler

import (
	"fmt"
	"log"
	"strconv"

	"github.com/jinnblue/chatroom-test/internal/logic"
	"github.com/jinnblue/chatroom-test/internal/proto"
	"github.com/jinnblue/chatroom-test/pkg/tcp"
)

type ServerHandle struct {
	prot tcp.Protocol
}

func NewServerHandle(prot tcp.Protocol) *ServerHandle {
	return &ServerHandle{prot: prot}
}

func (h *ServerHandle) OnConnect(c *tcp.TCPConn) bool {
	// new user
	user := logic.NewServerUser(c)
	c.SetExtraData(user)
	fmt.Printf("client:%d OnConnect: init user: %v\n", c.OnlineIdx, user)
	return true
}

func (h *ServerHandle) OnMessage(c *tcp.TCPConn, p tcp.Packet) bool {
	// log.Println("OnMessage client:", c.OnlineIdx)
	err := h.prot.Route(p, c.GetExtraData()) // 0:msg 1:*user
	if err != nil {
		log.Printf("client:%d OnMessage error: %v\n", c.OnlineIdx, err)
		return false
	}
	return true
}

func (h *ServerHandle) OnClose(c *tcp.TCPConn) {
	user := (c.GetExtraData()).(*logic.User)
	log.Printf("client:%d OnClose: %v\n", c.OnlineIdx, user)
	logic.RoomAdmin().Logout(user)
}

func CMLogin(param []interface{}) {
	// 0:msg 1:*user
	cmsg := param[0].(*proto.CMLogin)
	user := param[1].(*logic.User)

	resp := &proto.SMRespLogin{ErrCode: proto.NICK_NAME_EXIST}
	if logic.RoomAdmin().Login(cmsg.NickName, user) {
		user.Nickname = cmsg.NickName
		resp.ErrCode = proto.LOGIN_OK
	}
	user.AsyncSendMessage(resp)
}

func CMEnter(param []interface{}) {
	// 0:msg 1:*user
	cmsg := param[0].(*proto.CMEnter)
	user := param[1].(*logic.User)

	resp := &proto.SMRespEnter{ErrCode: proto.INVALID_ROOM_ID}
	if logic.RoomAdmin().EnterRoom(cmsg.RoomId, user) {
		user.RoomId = cmsg.RoomId
		resp.ErrCode = proto.ENTER_OK
	}
	user.AsyncSendMessage(resp)
}

func CMLeave(param []interface{}) {
	// 0:msg 1:*user
	// cmsg := param[0].(*proto.CMLeave)
	user := param[1].(*logic.User)

	resp := &proto.SMRespLeave{ErrCode: proto.NOT_IN_ROOM}
	if logic.RoomAdmin().LeaveRoom(user) {
		user.RoomId = 0
		resp.ErrCode = proto.LEAVE_OK
	}
	user.AsyncSendMessage(resp)
}

func CMChat(param []interface{}) {
	// 0:msg 1:*user
	cmsg := param[0].(*proto.CMChat)
	user := param[1].(*logic.User)

	smsg := &proto.SMChatContent{
		NickName: user.Nickname,
		Content:  cmsg.Content,
		SendTime: cmsg.SendTime,
	}
	smsg.BackupContent()
	logic.RoomAdmin().ChatInRoom(user, smsg)
}

func CMCommandGM(param []interface{}) {
	// 0:msg 1:*user
	cmsg := param[0].(*proto.CMCommandGM)
	user := param[1].(*logic.User)

	switch cmsg.CmdType {
	case proto.POPULAR:
		{
			id, err := strconv.Atoi(cmsg.Param)
			if err != nil {
				id = 0
			}
			word := logic.RoomAdmin().GetRoomPopularWord(uint32(id))
			smsg := &proto.SMPopularWord{
				TheWord: word,
			}
			user.AsyncSendMessage(smsg)
		}
	case proto.STATS:
		{
			s := logic.RoomAdmin().GetUserStats(cmsg.Param)
			smsg := &proto.SMUserStats{
				NickName: cmsg.Param,
				Stats:    s,
			}
			user.AsyncSendMessage(smsg)
		}
	}
}
