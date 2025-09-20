package ffmpegcmd

import (
	"fmt"
	"os"
	"os/exec"
)

func ImageWatermark2Video(inputVideo, watermarkImage string, x, y int, outputVideo string) error {
	// 构建 FFmpeg 命令参数
	args := []string{
		"-i", inputVideo, // 输入视频文件
		"-i", watermarkImage, // 水印图片文件
		"-filter_complex", fmt.Sprintf("overlay=%d:%d", x, y), // 叠加水印
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
