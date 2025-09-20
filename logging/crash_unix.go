// +build unix darwin

package logging

import (
    "log"
    "os"
    "syscall"
)

func CrashLog(file string) {
    f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
    if err != nil {
        log.Println(err)
        return
    }
    // Unix 下重定向标准错误输出
    syscall.Dup2(int(f.Fd()), syscall.Stderr)
    // 可选：也重定向 os.Stderr
    os.Stderr = f
}