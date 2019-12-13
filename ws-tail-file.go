package main

import (
	"flag"
	"github.com/waves-zhangyt/ws-tail-file/ws"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	flag.Parse()

	http.Handle(
		"/html/",
		http.StripPrefix(
			"/html/",
			http.FileServer(http.Dir("html")),
		),
	)

	http.HandleFunc("/tailFile", ws.ExampleTailFile)

	http.HandleFunc("/startCommand", ws.StartExampleCommand)
	http.HandleFunc("/commandOutput", ws.ExampleCommandOutput)

	log.Println("服务器启动， 监听 7979 端口。")
	server := &http.Server{
		Addr:         ":7979",
		ReadTimeout:  10 * time.Minute,
		WriteTimeout: 10 * time.Minute,
	}
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Fatal(err)
			return
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	for {
		select {
		case <-interrupt:
			log.Println("interrupted, system quit normally")
			return
		}
	}
}
