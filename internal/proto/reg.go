package proto

import (
	"github.com/jinnblue/chatroom-test/pkg/tcp/protocol"
)

func RegAllClientMsg(prot *protocol.GobProtocol) {
	prot.Register(&CMLogin{})
	prot.Register(&CMEnter{})
	prot.Register(&CMLeave{})
	prot.Register(&CMChat{})
	prot.Register(&CMCommandGM{})
}

func RegAllServerMsg(prot *protocol.GobProtocol) {
	prot.Register(&SMRespLogin{})
	prot.Register(&SMRespEnter{})
	prot.Register(&SMRespLeave{})
	prot.Register(&SMUserEnter{})
	prot.Register(&SMUserLeave{})
	prot.Register(&SMChatContent{})
	prot.Register(&SMUserStats{})
	prot.Register(&SMPopularWord{})
}
