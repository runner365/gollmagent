package ffmpegcmd

import (
	"fmt"
	"os"
	"os/exec"

	log "github.com/gollmagent/logging"
)

func SupportedTextColor() []string {
	return []string{
		"black", "white", "red", "green", "blue", "yellow", "cyan", "magenta",
		"gray", "grey", "darkred", "darkgreen", "darkblue", "darkyellow",
		"darkcyan", "darkmagenta", "lightgray", "lightgrey",
	}
}

func TextWatermark2Video(inputVideo, watermarkText string, x, y int, colorString string, outputVideo string) error {

	colors := SupportedTextColor()
	supported := false
	for _, c := range colors {
		if c == colorString {
			supported = true
			break
		}
	}
	if !supported {
		log.Errorf("Unsupported color: %s. Supported colors are: %v", colorString, colors)
		return fmt.Errorf("unsupported color: %s", colorString)
	}
	// 构建 FFmpeg 命令参数
	args := []string{
		"-i", inputVideo, // 输入视频文件
		"-vf", fmt.Sprintf("drawtext=text='%s':fontcolor=%s:fontsize=24:x=%d:y=%d", watermarkText, colorString, x, y), // 叠加文本水印
		"-codec:a", "copy", // 保持音频编码不变
		"-y",        // 覆盖输出文件
		outputVideo, // 输出视频文件
	}

	// 执行 FFmpeg 命令
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// log print ffmpeg command
	cmdDbg := "ffmpeg"
	for _, arg := range args {
		cmdDbg += fmt.Sprintf(" %s", arg)
	}
	fmt.Println("Executing command:", cmdDbg)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg execution failed: %v", err)
	}

	return nil
}
