package handler

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jinnblue/chatroom-test/internal/logic"
	"github.com/jinnblue/chatroom-test/internal/proto"
	"github.com/jinnblue/chatroom-test/pkg/tcp"
)

type ClientHandle struct {
	prot tcp.Protocol
	user *logic.User
}

func NewClientHandle(prot tcp.Protocol) *ClientHandle {
	return &ClientHandle{prot: prot}
}

func (h *ClientHandle) OnConnect(c *tcp.TCPConn) bool {
	fmt.Println("connect chatroom successed")

	nickname := getNickname()
	h.user = logic.NewClientUser(c, nickname)
	c.SetExtraData(h.user)

	err := c.AsyncSendPacket(&proto.CMLogin{
		NickName: nickname,
		SendTime: time.Now().Unix(),
	})
	return err == nil
}

func getNickname() string {
	var nickname string
	for len(nickname) <= 0 {
		fmt.Println("please enter your NickName：")
		fmt.Scanln(&nickname)
		// nickname = "jinnblue"
	}
	return nickname
}

const HELP_HINT = `命令列表:
			/popular [roomId]    显示10分钟内该房间词频最高的单词
			/stats [nickName]    显示 nickName 对应用户信息
			/leave               离开房间
			/exit                退出
			/help                显示命令`

const (
	CMD_POPULAR = "/popular"
	CMD_STATS   = "/stats"
	CMD_LEAVE   = "/leave"
	CMD_HELP    = "/help"
	CMD_EXIT    = "/exit"
)

func parseCmd(text string) (cmd, param string) {
	words := strings.Fields(text)
	switch len(words) {
	case 1:
		return strings.ToLower(words[0]), ""
	case 2:
		return strings.ToLower(words[0]), words[1]
	}
	return strings.ToLower(text), ""
}

func procEnterText(usr *logic.User) {
	var msgtext string
	scan := bufio.NewScanner(os.Stdin)
	for scan.Scan() {
		var cmsg tcp.Packet
		msgtext = scan.Text()
		if strings.IndexByte(msgtext, '/') == 0 {
			// client GM cmd
			cmd, param := parseCmd(msgtext)
			// fmt.Println("text:", msgtext, " cmd:", cmd, " param:", param)
			switch cmd {
			case CMD_POPULAR:
				_, err := strconv.Atoi(param)
				if err != nil {
					fmt.Println("roomId 必须为数字,示例: /popular [roomId]")
					continue
				}
				cmsg = &proto.CMCommandGM{
					CmdType: proto.POPULAR,
					Param:   param,
				}
				usr.AsyncSendMessage(cmsg)
			case CMD_STATS:
				if param == "" {
					fmt.Println("nickname 不可为空,示例: /stats [nickname]")
					continue
				}
				cmsg = &proto.CMCommandGM{
					CmdType: proto.STATS,
					Param:   param,
				}
				usr.AsyncSendMessage(cmsg)
			case CMD_LEAVE:
				cmsg = &proto.CMLeave{}
				usr.AsyncSendMessage(cmsg)
				return
			case CMD_HELP:
				fmt.Println(HELP_HINT)
			case CMD_EXIT:
				os.Exit(0)
			default:
				fmt.Println("不支持的指令")
			}
		} else {
			// client chat msg
			cmsg = &proto.CMChat{
				Content:  msgtext,
				SendTime: time.Now().Unix(),
			}
			usr.AsyncSendMessage(cmsg)
			// log.Printf("procEnterText text:%s msg:%v\n", msgtext, cmsg)
		}
	}
}

func (h *ClientHandle) OnMessage(c *tcp.TCPConn, p tcp.Packet) bool {
	err := h.prot.Route(p, c.GetExtraData()) // 0:msg 1:*user
	if err != nil {
		log.Printf("OnMessage error: %v\n", err)
		return false
	}
	return true
}

func (h *ClientHandle) OnClose(c *tcp.TCPConn) {
	user := (c.GetExtraData()).(*logic.User)
	log.Println("Disconnected: ", user)
}

func (h *ClientHandle) SendChatContent(c string) {
	cm := &proto.CMChat{
		Content:  c,
		SendTime: time.Now().Unix(),
	}
	h.user.AsyncSendMessage(cm)
}

func getRoomId() uint32 {
	var roomId uint32
	for roomId <= 0 {
		fmt.Println("please enter roomId：")
		fmt.Scanln(&roomId)
	}
	return roomId
}

func SMRespLogin(param []interface{}) {
	// 0:msg 1:*user
	smsg := param[0].(*proto.SMRespLogin)
	user := param[1].(*logic.User)
	switch smsg.ErrCode {
	case proto.LOGIN_OK:
		{
			fmt.Printf("SYSTEM: %s 登录成功\n", user.Nickname)
			fmt.Println(HELP_HINT)
			user.RoomId = getRoomId()
			user.AsyncSendMessage(&proto.CMEnter{
				RoomId: user.RoomId,
			})
		}
	case proto.NICK_NAME_EXIST:
		{
			fmt.Println("SYSTEM: 昵称已存在,请重新输入")
			//client reset nickname
			user.Nickname = getNickname()
			user.AsyncSendMessage(&proto.CMLogin{
				NickName: user.Nickname,
				SendTime: time.Now().Unix(),
			})
		}
	}
}

func SMRespEnter(param []interface{}) {
	// 0:msg 1:*user
	smsg := param[0].(*proto.SMRespEnter)
	user := param[1].(*logic.User)
	switch smsg.ErrCode {
	case proto.ENTER_OK:
		{
			fmt.Printf("SYSTEM: %s 欢迎进入聊天室[%d]\n", user.Nickname, user.RoomId)
			go procEnterText(user)
		}
	case proto.INVALID_ROOM_ID:
		{
			fmt.Println("SYSTEM: 无效的RoomId,请重新输入")
			//client reset roomId
			user.RoomId = getRoomId()
			user.AsyncSendMessage(&proto.CMEnter{
				RoomId: user.RoomId,
			})
		}
	}
}

func SMRespLeave(param []interface{}) {
	// 0:msg 1:*user
	smsg := param[0].(*proto.SMRespLeave)
	user := param[1].(*logic.User)
	switch smsg.ErrCode {
	case proto.LEAVE_OK, proto.INVALID_ROOM_ID:
		{
			fmt.Printf("SYSTEM: 已离开聊天室[%d],请选择要进入的聊天室\n", user.RoomId)
			user.RoomId = getRoomId()
			user.AsyncSendMessage(&proto.CMEnter{
				RoomId: user.RoomId,
			})
		}
	}
}

func SMUserEnter(param []interface{}) {
	// 0:msg 1:*user
	smsg := param[0].(*proto.SMUserEnter)
	fmt.Printf("SYSTEM: 用户 %s 加入了聊天室\n", smsg.NickName)
}

func SMUserLeave(param []interface{}) {
	// 0:msg 1:*user
	smsg := param[0].(*proto.SMUserLeave)
	fmt.Printf("SYSTEM: 用户 %s 离开了聊天室\n", smsg.NickName)
}

func SMChatContent(param []interface{}) {
	// 0:msg 1:*user
	smsg := param[0].(*proto.SMChatContent)
	// user := param[1].(*logic.User)
	fmt.Printf("%s: %s\n", smsg.NickName, smsg.Content)
}

func SMUserStats(param []interface{}) {
	// 0:msg 1:*user
	smsg := param[0].(*proto.SMUserStats)
	// user := param[1].(*logic.User)
	// fmt.Printf("%s: %s\n", smsg.NickName, smsg.Stats)
	fmt.Printf("%s\n", smsg.Stats)
}

func SMPopularWord(param []interface{}) {
	// 0:msg 1:*user
	smsg := param[0].(*proto.SMPopularWord)
	// user := param[1].(*logic.User)
	fmt.Printf("%s\n", smsg.TheWord)
}
