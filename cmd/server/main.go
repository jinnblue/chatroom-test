package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/jinnblue/chatroom-test/internal/handler"
	"github.com/jinnblue/chatroom-test/internal/logic"
	"github.com/jinnblue/chatroom-test/internal/proto"
	"github.com/jinnblue/chatroom-test/pkg/tcp"
	"github.com/jinnblue/chatroom-test/pkg/tcp/protocol"
)

var (
	addr      string
	cfgPath   string
	srvHandle tcp.Handler
	srvParser tcp.PacketParser
)

func main() {
	flag.StringVar(&addr, "addr", "0.0.0.0:20000", "IP:Port address of chatrooms listen on.")
	flag.StringVar(&cfgPath, "config", "", "config path of blackwords.")
	flag.Parse()

	logic.InitActrie(cfgPath)
	fmt.Printf("chatrooms server start on:%s \n", addr)

	f, _ := os.OpenFile("cpu.pprof", os.O_CREATE|os.O_RDWR, 0644)
	defer f.Close()
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	srv := tcp.NewTCPServer(addr, tcp.NewTCPOption(srvHandle, srvParser, tcp.WithSendChanLimit(100), tcp.WithRecvChanLimit(20)))
	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}
		defer srv.Close()
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("Signal: ", <-sigChan)
	logic.RoomAdmin().Close()
}

func init() {
	protGob := protocol.NewGobProtocol()

	proto.RegAllServerMsg(protGob)
	protGob.RegisterAndHandle(&proto.CMLogin{}, handler.CMLogin)
	protGob.RegisterAndHandle(&proto.CMEnter{}, handler.CMEnter)
	protGob.RegisterAndHandle(&proto.CMLeave{}, handler.CMLeave)
	// server chat msg
	protGob.RegisterAndHandle(&proto.CMChat{}, handler.CMChat)
	// server GM cmd
	protGob.RegisterAndHandle(&proto.CMCommandGM{}, handler.CMCommandGM)

	srvHandle = handler.NewServerHandle(protGob)
	srvParser = tcp.NewHeaderPacketParser(protGob)
}
