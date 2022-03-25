package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jinnblue/chatroom-test/internal/handler"
	"github.com/jinnblue/chatroom-test/internal/proto"
	"github.com/jinnblue/chatroom-test/pkg/tcp"
	"github.com/jinnblue/chatroom-test/pkg/tcp/protocol"
)

var (
	addr         string
	clientHandle *handler.ClientHandle
	clientParser tcp.PacketParser
)

func main() {
	flag.StringVar(&addr, "addr", "127.0.0.1:20000", "IP:Port address of chatroom to join.")
	flag.Parse()

	fmt.Println("connect chatroom on:", addr)
	opt := tcp.NewTCPOption(clientHandle, clientParser)
	conn := tcp.NewTCPClient(addr, 1, opt)
	conn.Start()
	defer conn.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("Signal: ", <-sigChan)
}

func init() {
	protGob := protocol.NewGobProtocol()

	proto.RegAllClientMsg(protGob)
	protGob.RegisterAndHandle(&proto.SMRespLogin{}, handler.SMRespLogin)
	protGob.RegisterAndHandle(&proto.SMRespEnter{}, handler.SMRespEnter)
	protGob.RegisterAndHandle(&proto.SMRespLeave{}, handler.SMRespLeave)
	protGob.RegisterAndHandle(&proto.SMUserEnter{}, handler.SMUserEnter)
	protGob.RegisterAndHandle(&proto.SMUserLeave{}, handler.SMUserLeave)
	// client reg chat msg
	protGob.RegisterAndHandle(&proto.SMChatContent{}, handler.SMChatContent)
	// client reg GM cmd
	protGob.RegisterAndHandle(&proto.SMUserStats{}, handler.SMUserStats)
	protGob.RegisterAndHandle(&proto.SMPopularWord{}, handler.SMPopularWord)

	clientHandle = handler.NewClientHandle(protGob)
	clientParser = tcp.NewHeaderPacketParser(protGob)
}
