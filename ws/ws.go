package ws

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"ws-tail-file/tailfile"
)

var upgrader = websocket.Upgrader{} // use default options

func TailFile(w http.ResponseWriter, r *http.Request) {
	filePath := r.FormValue("filePath")

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	tf := tailfile.NewTailFile(filePath, true)
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
