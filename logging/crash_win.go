//go:build windows
// +build windows

package logging

import (
	"log"
	"os"
	"runtime"
)

func CrashLog(file string) {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Println(err)
		return
	}
	os.Stderr = f

	go func() {
		if r := recover(); r != nil {
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, true)
			f.WriteString("panic: ")
			f.WriteString(log.Prefix())
			f.WriteString("\n")
			f.WriteString(string(buf[:n]))
		}
	}()
}
