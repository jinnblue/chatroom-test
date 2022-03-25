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
	num          int
	clientHandle tcp.Handler
	clientParser tcp.PacketParser
)

func main() {
	flag.StringVar(&addr, "addr", "127.0.0.1:20000", "IP:Port address of chatroom to join.")
	flag.IntVar(&num, "num", 100, "benchmark client num.")
	flag.Parse()

	fmt.Println("connect chatroom on:", addr)
	opt := tcp.NewTCPOption(clientHandle, clientParser, tcp.WithRecvChanLimit(1024), tcp.WithSendChanLimit(256))
	conn := tcp.NewTCPClient(addr, num, opt)
	conn.Start()
	defer conn.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("Signal: ", <-sigChan)

	close(handler.PrintCloseChan)
	close(handler.PrintChan)
}

func init() {
	protGob := protocol.NewGobProtocol()

	proto.RegAllClientMsg(protGob)
	protGob.RegisterAndHandle(&proto.SMRespLogin{}, handler.SMRespLoginBench)
	protGob.RegisterAndHandle(&proto.SMRespEnter{}, handler.SMRespEnterBench)
	protGob.RegisterAndHandle(&proto.SMRespLeave{}, handler.SMRespLeaveBench)
	protGob.RegisterAndHandle(&proto.SMUserEnter{}, handler.SMUserEnterBench)
	protGob.RegisterAndHandle(&proto.SMUserLeave{}, handler.SMUserLeaveBench)
	// client reg chat msg
	protGob.RegisterAndHandle(&proto.SMChatContent{}, handler.SMChatContentBench)
	// client reg GM cmd
	protGob.RegisterAndHandle(&proto.SMUserStats{}, handler.SMUserStatsBench)
	protGob.RegisterAndHandle(&proto.SMPopularWord{}, handler.SMPopularWordBench)

	clientHandle = handler.NewClientBenchHandle(protGob)
	clientParser = tcp.NewHeaderPacketParser(protGob)
}
