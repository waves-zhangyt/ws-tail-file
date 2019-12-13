package ws

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	imlog "github.com/waves-zhangyt/ws-tail-file/imoutput"
	"io"
	"net/http"
	"time"
)

func StartExampleCommand(w http.ResponseWriter, r *http.Request) {
	taskId := r.FormValue("taskId")

	cmdstr := "echo `date`\n" +
		"sleep 5\n echo `date`\n" +
		"sleep 2\n echo `date`\n" +
		"sleep 3\n echo `date`\n" +
		"sleep 1\n echo `date`\n" +
		"sleep 2\n echo `date`\n" +
		"sleep 3\n echo `date`\n" +
		"sleep 2\n echo `date`\n" +
		"sleep 1\n echo `date`\n"

	var commandOutMonitor imlog.CommandOutMonitor
	commandOutMonitor.StartMonitorOutput()
	imlog.CommandOutMonitors.SetDefault(taskId, &commandOutMonitor)

	go func() {
		err := imlog.LocalCommandTimeout(cmdstr, &commandOutMonitor, 180)
		if err != nil {
			glog.Errorf("run err : %v", err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	m := make(map[string]string)

	m["msg"] = "started"
	b, _ := json.Marshal(m)
	io.WriteString(w, string(b))

}

func ExampleCommandOutput(w http.ResponseWriter, r *http.Request) {
	taskId := r.FormValue("taskId")
	v, exist := imlog.CommandOutMonitors.Get(taskId)
	if !exist || v == nil {
		glog.Warningf("task not found: %v", taskId)
		return
	}
	commandOutMonitor := v.(*imlog.CommandOutMonitor)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		glog.Infof("upgrade:", err)
		return
	}
	defer conn.Close()

	glog.Infof("start read output: %v", taskId)
	ch := make(chan string)
	commandOutMonitor.FetchOutputToChannel(ch)
	for {
		select {
		case line, ok := <-ch:
			if !ok {
				glog.Infof("im log over, taskId: %v", taskId)
				return
			}
			err := conn.WriteMessage(websocket.TextMessage, []byte(line))
			if err != nil {
				glog.Infof("write mesage err(web socket session over):", err)
				return
			}
		case <-time.After(60 * time.Second):
			glog.Infof("read ch timeout, return")
			return
		}
	}

}
