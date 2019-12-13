package ws

import (
	"github.com/gorilla/websocket"
	"github.com/waves-zhangyt/ws-tail-file/tailfile"
	"log"
	"net/http"
)

var upgrader = websocket.Upgrader{} // use default options

func ExampleTailFile(w http.ResponseWriter, r *http.Request) {
	filePath := r.FormValue("filePath")

	TailFile(filePath, false, w, r)
}

func TailFile(filePath string, fromBottom bool, w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	tf := tailfile.NewTailFile(filePath, true, false)
	if tf == nil {
		c.WriteMessage(websocket.TextMessage, []byte("file not exists"))
		c.Close()
		return
	}
	ch := make(chan string)
	tf.StartScan(ch)
	defer tf.StopScan()

	over := make(chan bool)

	go func() {
		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				over <- true
				log.Printf("client disconnection: %v", err)
				return
			}
		}
	}()

	for {
		select {
		case line, ok := <-ch:
			if !ok {
				return
			}
			err := c.WriteMessage(websocket.TextMessage, []byte(line+"\n"))
			if err != nil {
				log.Println("write mesage err(web socket session over):", err)
				return
			}
		case <-over:
			log.Println("websocket session over")
			tf.StopScan()
			return
		}
	}
}
