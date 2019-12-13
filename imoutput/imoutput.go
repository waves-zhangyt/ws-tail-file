package imoutput

import (
	"bytes"
	"context"
	"github.com/golang/glog"
	"github.com/patrickmn/go-cache"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// taskId -> CommandOutMonitor
var CommandOutMonitors = cache.New(30*time.Minute, 1*time.Hour)

type CommandOutMonitor struct {
	StdoutLines []string
	StderrLines []string
	Stdout      bytes.Buffer
	Stderr      bytes.Buffer
	Done        bool
	DoneMutex   sync.Mutex
}

func (c *CommandOutMonitor) GetStderrString() string {
	var buf strings.Builder
	for _, line := range c.StderrLines {
		buf.WriteString(line)
	}

	return buf.String()
}

func (c *CommandOutMonitor) GetStdoutString() string {
	var buf strings.Builder
	for _, line := range c.StdoutLines {
		buf.WriteString(line)
	}

	return buf.String()
}

func (c *CommandOutMonitor) FetchOutputToChannel(ch chan string) {
	go func() {
		defer func() {
			//relay close
			time.Sleep(3 * time.Second)
			close(ch)
		}()

		i := 0
		for !c.Done || i < len(c.StdoutLines) {
			time.Sleep(200 * time.Millisecond)

			if i >= len(c.StdoutLines) {
				continue
			}
			ch <- c.StdoutLines[i]
			i++
		}
		glog.Infof("stdout monitor over")
	}()

	go func() {
		i := 0
		for !c.Done || i < len(c.StderrLines) {
			time.Sleep(200 * time.Millisecond)

			if i >= len(c.StderrLines) {
				continue
			}
			ch <- c.StdoutLines[i]
			i++
		}
		glog.Infof("stderr monitor over")
	}()
}

func (c *CommandOutMonitor) FetchOutputToWriter(writer io.Writer) {
	go func() {
		i := 0
		for !c.Done || i < len(c.StdoutLines) {
			time.Sleep(200 * time.Millisecond)

			if i >= len(c.StdoutLines) {
				continue
			}

			_, err := writer.Write([]byte(c.StdoutLines[i]))
			if err != nil {
				glog.Errorf("write stdout to target err: %v", err)
			}
			i++
		}
		glog.Infof("stdout monitor over")
	}()

	go func() {
		i := 0
		for !c.Done || i < len(c.StderrLines) {
			time.Sleep(200 * time.Millisecond)

			if i >= len(c.StderrLines) {
				continue
			}

			_, err := writer.Write([]byte(c.StderrLines[i]))
			if err != nil {
				glog.Errorf("write stderr to target err: %v", err)
			}
			i++
		}
		glog.Infof("stderr monitor over")
	}()
}

func (c *CommandOutMonitor) IsDone() bool {
	c.DoneMutex.Lock()
	defer c.DoneMutex.Unlock()
	return c.Done
}

func (c *CommandOutMonitor) SetDone(done bool) {
	c.DoneMutex.Lock()
	defer c.DoneMutex.Unlock()
	c.Done = done
}

func (c *CommandOutMonitor) StartMonitorOutput() {
	go func() {
		for !c.IsDone() {
			time.Sleep(20 * time.Millisecond)
			line, err := c.Stdout.ReadString('\n')
			if line != "" {
				c.StdoutLines = append(c.StdoutLines, line)
				line, err = c.Stdout.ReadString('\n')
			}
			if err != nil && err != io.EOF {
				glog.Errorf("read monitor stdout err: %v", err)
			}
		}
	}()
	go func() {
		for !c.IsDone() {
			time.Sleep(20 * time.Millisecond)
			line, err := c.Stderr.ReadString('\n')
			if line != "" {
				c.StderrLines = append(c.StderrLines, line)
				line, err = c.Stderr.ReadString('\n')
			}
			if err != nil && err != io.EOF {
				glog.Errorf("read monitor stdout err: %v", err)
			}
		}
	}()
}

func LocalCommandTimeout(cmd string, commandOutMonitor *CommandOutMonitor, timeout int) error {
	ctxt, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer func() {
		cancel()
		time.Sleep(5 * time.Second)
		commandOutMonitor.SetDone(true)
	}()

	command := exec.CommandContext(ctxt, "bash", "-c", cmd)
	command.Stdout = &(commandOutMonitor.Stdout)
	command.Stderr = &(commandOutMonitor.Stderr)
	if err := command.Run(); err != nil {
		if ctxt.Err() == context.DeadlineExceeded {
			log.Printf("command timeout: %v\n", err)
			return err
		}
		glog.Errorf("执行命令出错 %v", err)
		return err
	}

	return nil
}
