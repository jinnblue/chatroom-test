package logic

import (
	"container/ring"

	"github.com/jinnblue/chatroom-test/internal/proto"
)

type OfflineMsg struct {
	recentRing *ring.Ring // 保存所有用户最近n条消息
}

func NewOfflineMsg(max int) *OfflineMsg {
	return &OfflineMsg{
		recentRing: ring.New(max),
	}
}

func (o *OfflineMsg) Save(msg *proto.SMChatContent) {
	o.recentRing.Value = msg
	o.recentRing = o.recentRing.Next()
}

func (o *OfflineMsg) Send(user *User) {
	o.recentRing.Do(func(val interface{}) {
		msg, ok := val.(*proto.SMChatContent)
		if ok {
			user.AsyncSendMessage(msg)
		}
	})
}
