package tailfile

import (
	"log"
	"os"
	"strings"
	"time"
)

type TailFile struct {
	File    string
	Offset  int64
	LineBuf strings.Builder
	Over    bool
}

func NewTailFile(filePath string, end, bottom bool) *TailFile {
	var tf TailFile
	tf.File = filePath
	tf.LineBuf = strings.Builder{}

	// tail from file head
	if !end {
		tf.Offset = 0
	} else {
		// tail from file end sub some lines
		info, err := os.Stat(tf.File)
		if err != nil {
			log.Printf("get file size err: %v", err)
			return nil
		}

		if info.Size() < 1600 {
			tf.Offset = 0
		} else {
			tf.findTheNextLineNew(info.Size(), 2000)
		}

		// tail from the read end bottom
		if bottom {
			tf.Offset = info.Size()
		}
	}

	tf.Over = false

	return &tf
}

func (tf *TailFile) findTheNextLineNew(fileSize int64, startBefore int) {
	f, err := os.Open(tf.File)
	if err != nil {
		log.Printf("open file error: %v", err)
		return
	}
	defer f.Close()

	start := fileSize - int64(startBefore)
	f.Seek(int64(start), 0)
	for i := 0; i < startBefore; i++ {
		buf := make([]byte, 1)
		n, err := f.Read(buf)
		if err != nil || n != 1 {
			log.Println("find the start new line err: ", err)
			tf.Offset = fileSize
			return
		}
		if buf[0] == '\n' {
			tf.Offset = start + int64(i) + 1
			log.Printf("find new line at %v", i)
			return
		}
	}

	// can't find, set at the end
	tf.Offset = fileSize
}

// readline or nil
func (tf *TailFile) ReadLine() *string {
	f, err := os.Open(tf.File)
	if err != nil {
		log.Printf("open file error: %v", err)
		return nil
	}
	defer f.Close()

	for {
		info, err := f.Stat()
		if err != nil {
			log.Println(err)
		}
		if tf.Offset >= info.Size() {
			return nil
		}

		f.Seek(tf.Offset, 0)
		buf := make([]byte, 128)
		n, err := f.Read(buf)
		if err != nil {
			log.Printf("may see eof: %v", err)
			return nil
		}

		m := 0
		for i := 0; i < n; i++ {
			if buf[i] == '\n' {
				tf.LineBuf.Write(buf[m:i])
				line := tf.LineBuf.String()
				tf.LineBuf.Reset()
				m = i + 1

				tf.Offset += int64(m)
				return &line
			}
		}
		tf.LineBuf.Write(buf[m:n])
		tf.Offset += int64(n)
	}
}

// start goroutine
func (tf *TailFile) StartScan(ch chan string) {
	log.Printf("开启日志扫描: %v", *tf)
	go func() {
		for {
			if tf.Over {
				log.Printf("scan over: %v", tf.File)
				return
			}

			start := time.Now()

			info, err := os.Stat(tf.File)
			if err != nil {
				log.Printf("stat file error: %v", err)
			}

			var line *string
			if tf.Offset < info.Size() {
				line = tf.ReadLine()
				if line != nil {
					ch <- *line
				}
			}

			// no sleep when have content
			if line != nil {
				continue
			}

			interval := time.Second - (time.Now().Sub(start))
			if interval > 0 {
				time.Sleep(interval)
			}
		}
	}()
}

// indicate to stop the scanning
func (tf *TailFile) StopScan() {
	tf.Over = true
}
