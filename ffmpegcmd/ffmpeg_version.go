package ffmpegcmd

import (
	"bytes"
	"context"
	"os/exec"
	"regexp"
	"time"
)

// 提取版本号，例如 "8.0" 或 "n8.0-3-g08a81b090b"
var versionRe = regexp.MustCompile(`ffmpeg\s+(?:version\s+)?([0-9n][0-9a-zA-Z.-]*)`)

// GetFFmpegVersion 返回本地 ffmpeg 的版本号；出错返回空串
func GetFFmpegVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg", "-version")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out // ffmpeg 把版本信息打在 stderr

	_ = cmd.Run() // 总会返回 exit code 1，忽略即可

	m := versionRe.FindStringSubmatch(out.String())
	if len(m) > 1 {
		return m[1]
	}
	return ""
}
