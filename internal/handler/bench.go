package handler

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/jinnblue/chatroom-test/internal/logic"
	"github.com/jinnblue/chatroom-test/internal/proto"
	"github.com/jinnblue/chatroom-test/pkg/pathmap"
	"github.com/jinnblue/chatroom-test/pkg/tcp"
)

var (
	SendChatDuration = 1000 * time.Millisecond
	shitWords        []string
)

func init() {
	abPath := pathmap.GetCurrentAbPath()
	cfgPath := filepath.Join(abPath, "../../", logic.DEFAULT_FILTER_FILE)
	f, err := os.Open(cfgPath)
	if err != nil {
		log.Fatal("benchmark client handle init err:", err)
	}

	shitWords = make([]string, 0, 512)
	r := bufio.NewReader(f)
	for {
		word, err := r.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}
		shitWords = append(shitWords, word)
	}
}

type ClientBenchHandle struct {
	prot tcp.Protocol
	user *logic.User
}

func NewClientBenchHandle(prot tcp.Protocol) *ClientBenchHandle {
	return &ClientBenchHandle{prot: prot}
}

func (h *ClientBenchHandle) OnConnect(c *tcp.TCPConn) bool {
	fmt.Printf("connect chatroom successed OnlineIdx: %d\n", c.OnlineIdx)

	globalIdx = int64(c.OnlineIdx)
	nickname := fmt.Sprintf("Client_%5.5d", c.OnlineIdx)
	h.user = logic.NewClientUser(c, nickname)
	c.SetExtraData(h.user)

	startPrint()

	err := c.AsyncSendPacket(&proto.CMLogin{
		NickName: nickname,
		SendTime: time.Now().Unix(),
	})
	return err == nil
}

func (h *ClientBenchHandle) OnMessage(c *tcp.TCPConn, p tcp.Packet) bool {
	err := h.prot.Route(p, c.GetExtraData()) // 0:msg 1:*user
	if err != nil {
		fmt.Printf("OnMessage error: %v\n", err)
		return false
	}
	return true
}

func (h *ClientBenchHandle) OnClose(c *tcp.TCPConn) {
	user := (c.GetExtraData()).(*logic.User)
	fmt.Println("Disconnected: ", user)
}

func (h *ClientBenchHandle) SendChatContent(c string) {
	cm := &proto.CMChat{
		Content:  c,
		SendTime: time.Now().Unix(),
	}
	h.user.AsyncSendMessage(cm)
}

func SMRespLoginBench(param []interface{}) {
	// 0:msg 1:*user
	smsg := param[0].(*proto.SMRespLogin)
	user := param[1].(*logic.User)
	switch smsg.ErrCode {
	case proto.LOGIN_OK:
		{
			fmt.Printf("SYSTEM: %s 登录成功\n", user.Nickname)
			user.RoomId = uint32(rand.Intn(10))
			user.AsyncSendMessage(&proto.CMEnter{
				RoomId: user.RoomId,
			})
		}
	case proto.NICK_NAME_EXIST:
		{
			fmt.Println("SYSTEM: 昵称已存在,请重新输入")
			//client reset nickname
			user.Nickname = fmt.Sprintf("ClientReset_%3.3d", atomic.AddInt64(&globalCID, 1))
			user.AsyncSendMessage(&proto.CMLogin{
				NickName: user.Nickname,
				SendTime: time.Now().Unix(),
			})
		}
	}
}

func SMRespEnterBench(param []interface{}) {
	// 0:msg 1:*user
	smsg := param[0].(*proto.SMRespEnter)
	user := param[1].(*logic.User)
	switch smsg.ErrCode {
	case proto.ENTER_OK:
		{
			fmt.Printf("SYSTEM: %s 欢迎进入聊天室[%d]\n", user.Nickname, user.RoomId)
			go func(u *logic.User) {
				for {
					l := len(shitWords)
					all := rand.Intn(10) + 1
					var say string
					for i := 0; i < all; i++ {
						n := rand.Intn(l)
						say = say + " " + shitWords[n]
					}
					cmsg := &proto.CMChat{
						Content:  fmt.Sprintf("Client benchmark test say %d %s", atomic.AddInt64(&globalCID, 1), say),
						SendTime: time.Now().Unix(),
					}
					u.AsyncSendMessage(cmsg)
					tmp := time.Duration(rand.Int63n(500)) * time.Millisecond
					time.Sleep(SendChatDuration + tmp)
				}
			}(user)
		}
	case proto.INVALID_ROOM_ID:
		{
			fmt.Println("SYSTEM: 无效的RoomId,请重新输入")
			//client reset roomId
			user.RoomId = 1 + uint32(rand.Intn(5))
			user.AsyncSendMessage(&proto.CMEnter{
				RoomId: user.RoomId,
			})
		}
	}
}

func SMRespLeaveBench(param []interface{}) {
	// 0:msg 1:*user
	// smsg := param[0].(*proto.SMRespLeave)
	// user := param[1].(*logic.User)
}

var (
	globalCID      int64 = rand.Int63()
	globalIdx      int64
	PrintChan      chan *proto.SMChatContent
	PrintCloseChan chan struct{}
)

func SMUserEnterBench(param []interface{}) {
	// 0:msg 1:*user
	// smsg := param[0].(*proto.SMUserEnter)
	// fmt.Printf("SYSTEM: 用户 %s 加入了聊天室\n", smsg.NickName)
}

func SMUserLeaveBench(param []interface{}) {
	// 0:msg 1:*user
	// smsg := param[0].(*proto.SMUserLeave)
	// fmt.Printf("SYSTEM: 用户 %s 离开了聊天室\n", smsg.NickName)
}

func SMChatContentBench(param []interface{}) {
	// 0:msg 1:*user
	smsg := param[0].(*proto.SMChatContent)
	// user := param[1].(*logic.User)
	asyncPrint(smsg)
	// log.Printf("%s: %s\n", smsg.NickName, smsg.Content)
}

func SMUserStatsBench(param []interface{}) {
	// 0:msg 1:*user
	smsg := param[0].(*proto.SMUserStats)
	// user := param[1].(*logic.User)
	// fmt.Printf("%s: %s\n", smsg.NickName, smsg.Stats)
	fmt.Printf("%s\n", smsg.Stats)
}

func SMPopularWordBench(param []interface{}) {
	// 0:msg 1:*user
	smsg := param[0].(*proto.SMPopularWord)
	// user := param[1].(*logic.User)
	fmt.Printf("%s\n", smsg.TheWord)
}

func init() {
	PrintChan = make(chan *proto.SMChatContent, 4096)
	PrintCloseChan = make(chan struct{})
}

func startPrint() {
	go func() {
		var lasttime int64 = time.Now().UnixNano()
		di := globalIdx
		diff := di * 3 * time.Second.Nanoseconds()
		for {
			select {
			case smsg := <-PrintChan:
				now := time.Now().UnixNano()
				if (now - lasttime) > diff {
					lasttime = now
					log.Printf("[%d] %s: %s\n", di, smsg.NickName, smsg.Content)
				}
				smsg = nil
			case <-PrintCloseChan:
				return
			}
		}
	}()
}

func asyncPrint(msg *proto.SMChatContent) {
	PrintChan <- msg
}
