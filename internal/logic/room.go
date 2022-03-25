package logic

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jinnblue/chatroom-test/internal/proto"
	"github.com/jinnblue/chatroom-test/pkg/acascii"
	"github.com/jinnblue/chatroom-test/pkg/pathmap"
	"github.com/jinnblue/chatroom-test/pkg/popular"
)

const (
	ROOM_NUM            = 5
	MSG_QUEUE_LEN       = 40960
	MAX_OFFLINE_MSG     = 50
	MAX_POPULAR_DURA    = 10 * time.Minute
	DEFAULT_FILTER_FILE = "internal/data/list.txt"
)

// RoomManager 聊天室管理器
type RoomManager struct {
	allUsersMap sync.Map // map[string]*User 所有用户
	roomsMap    sync.Map // map[int]*Room 所有聊天室
}

func (rm *RoomManager) CreateRoom(num int) {
	for i := 0; i < num; i++ {
		room := newChatRoom()
		_, exist := rm.roomsMap.LoadOrStore(room.ident, room)
		if exist {
			panic("CreateRoom error: roomId Duplicated!")
		}
	}
}

// Login 登录,昵称必须唯一
func (rm *RoomManager) Login(nickname string, usr *User) bool {
	_, exist := rm.allUsersMap.LoadOrStore(nickname, usr)
	return !exist
}

// Logout 登出
func (rm *RoomManager) Logout(usr *User) bool {
	if rm.LeaveRoom(usr) {
		rm.allUsersMap.Delete(usr.Nickname)
	}
	return false
}

// EnterRoom 进入聊天室
func (rm *RoomManager) EnterRoom(roomid uint32, usr *User) bool {
	val, has := rm.roomsMap.Load(roomid)
	if has {
		room, ok := val.(*Room)
		if ok {
			room.UserEntering(usr)
			return true
		}
	}
	return false
}

// LeaveRoom 离开聊天室
func (rm *RoomManager) LeaveRoom(usr *User) bool {
	if usr.RoomId == 0 {
		return false
	}
	val, has := rm.roomsMap.Load(usr.RoomId)
	if has {
		room, ok := val.(*Room)
		if ok {
			room.UserLeaving(usr)
			return true
		}
	}
	return false
}

// ChatInRoom 聊天
func (rm *RoomManager) ChatInRoom(usr *User, msg *proto.SMChatContent) bool {
	if usr.RoomId == 0 {
		return false
	}
	val, has := rm.roomsMap.Load(usr.RoomId)
	if has {
		room, ok := val.(*Room)
		if ok {
			room.Broadcast(usr, msg)
			return true
		}
	}
	return false
}

// GetRoomPopularWord 根据房间ID获取最高频单词
func (rm *RoomManager) GetRoomPopularWord(roomid uint32) string {
	val, has := rm.roomsMap.Load(roomid)
	if has {
		room, ok := val.(*Room)
		if ok {
			return room.GetPopularWord(MAX_POPULAR_DURA)
		}
	}
	return ""
}

// GetUserStats 根据玩家昵称获取用户信息
func (rm *RoomManager) GetUserStats(nickname string) string {
	val, has := rm.allUsersMap.Load(nickname)
	if has {
		usr, ok := val.(*User)
		if ok {
			diff := time.Now().UTC().Sub(usr.EnterAt)
			secs := diff / time.Second
			return fmt.Sprintf("LoginAt: %s  Online: %ds  RoomId: %d", usr.EnterAt, secs, usr.RoomId)
		}
	}
	return ""
}

func (rm *RoomManager) Close() {
	rm.roomsMap.Range(func(id, val interface{}) bool {
		room, ok := val.(*Room)
		if ok {
			room.Close()
		}
		return true
	})
}

// MessageBuff 消息缓存
type MessageBuff struct {
	buff   []byte
	srcMsg *proto.SMChatContent
}

// Room 单个聊天室
type Room struct {
	ident      uint32
	usersMap   sync.Map // map[string]*User
	closeChan  chan struct{}
	popular    *popular.MostPopularWord
	offlineMsg *OfflineMsg

	enteringChannel chan *User
	leavingChannel  chan *User
	messageChannel  chan *MessageBuff
}

var globalIdent uint32 = 0

// newChatRoom 创建并启动聊天室
func newChatRoom() *Room {
	r := &Room{
		ident:           atomic.AddUint32(&globalIdent, 1),
		usersMap:        sync.Map{},
		closeChan:       make(chan struct{}),
		popular:         popular.NewMostPopularWord(MAX_POPULAR_DURA),
		offlineMsg:      NewOfflineMsg(MAX_OFFLINE_MSG),
		enteringChannel: make(chan *User),
		leavingChannel:  make(chan *User),
		messageChannel:  make(chan *MessageBuff, MSG_QUEUE_LEN),
	}
	go r.Start()
	return r
}

func (r *Room) GetPopularWord(past time.Duration) string {
	return r.popular.GetTopWord(past)
}

func (r *Room) UserEntering(usr *User) {
	r.enteringChannel <- usr
}

func (r *Room) UserLeaving(usr *User) {
	r.leavingChannel <- usr
}

func (r *Room) Broadcast(usr *User, msg *proto.SMChatContent) {
	//xTODO content filter
	msg.Content = trie.Filter(msg.Content)

	buf, err := usr.BuildMessageBuf(msg)
	if err != nil {
		log.Println("entering BuildMessageBuf error:", err)
		return
	}
	if len(r.messageChannel) >= MSG_QUEUE_LEN {
		log.Println("Room messageChannel is full")
	}
	r.messageChannel <- &MessageBuff{buff: buf, srcMsg: msg}
}

func (r *Room) broadsend(buf []byte, except string) {
	r.usersMap.Range(func(name, val interface{}) bool {
		user, ok := val.(*User)
		if ok && (user.Nickname != except) {
			user.AsyncSendBuff(buf)
		}
		return true
	})
}

func (r *Room) Start() {
	for {
		select {
		case <-r.closeChan:
			log.Println("Room go Closed")
			return
		case user := <-r.enteringChannel: // 新进入
			{
				r.usersMap.LoadOrStore(user.Nickname, user)

				// 发送离线消息
				r.offlineMsg.Send(user)

				// 通知其他用户
				smsg := &proto.SMUserEnter{
					NickName: user.Nickname,
					SendTime: time.Now().Unix(),
				}
				buf, err := user.BuildMessageBuf(smsg)
				if err != nil {
					log.Println("entering BuildMessageBuf error:", err)
					return
				}
				r.broadsend(buf, user.Nickname)
			}
		case user := <-r.leavingChannel: // 离开
			{
				r.usersMap.Delete(user.Nickname)

				// 通知其他用户
				smsg := &proto.SMUserLeave{
					NickName: user.Nickname,
					SendTime: time.Now().Unix(),
				}
				buf, err := user.BuildMessageBuf(smsg)
				if err != nil {
					log.Println("entering BuildMessageBuf error:", err)
					return
				}
				r.broadsend(buf, user.Nickname)
			}
		case m := <-r.messageChannel: // 广播
			{
				words := strings.Fields(m.srcMsg.Content)
				for _, w := range words {
					r.popular.Record(w)
				}

				r.broadsend(m.buff, m.srcMsg.NickName)

				// 离线消息保存
				r.offlineMsg.Save(m.srcMsg)
			}
		}
	}
}

func (r *Room) Close() {
	r.closeChan <- struct{}{}
}

var (
	rm     *RoomManager
	raonce sync.Once
	trie   *acascii.ACTrie
)

func RoomAdmin() *RoomManager {
	raonce.Do(func() {
		rm = new(RoomManager)
		rm.CreateRoom(ROOM_NUM)
	})
	return rm
}

func InitActrie(cfgpath string) {
	if cfgpath == "" {
		abPath := pathmap.GetCurrentAbPath()
		cfgpath = filepath.Join(abPath, "../../", DEFAULT_FILTER_FILE)
	}
	trie = acascii.NewACTrieFromFile(cfgpath)
	log.Println("actrie load black words from:", cfgpath)
}
