package ffmpegcmd

import (
	"bytes"
	"os/exec"
	"regexp"
	"strings"
)

// GetFFmpegConfig 返回 ffmpeg 编译配置参数的字符串切片
func GetFFmpegConfig() ([]string, error) {
	cmd := exec.Command("ffmpeg", "-version")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	// 正则：configuration: 后面直到行尾
	re := regexp.MustCompile(`(?m)^configuration:\s*(.+)$`)
	m := re.FindStringSubmatch(out.String())
	if len(m) < 2 {
		return nil, nil // 没找到配置行
	}

	// 按空格切分并去掉空字符串
	fields := strings.Fields(m[1])
	return fields, nil
}
