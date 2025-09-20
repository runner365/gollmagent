package ffmpegcmd

import (
	"fmt"
	"os/exec"

	log "github.com/gollmagent/logging"
)

func GenPictureFromVideoBaseOnIframe(inputVideo string, outputDir string) error {
	// ffmpeg cmd: gen picture from video based on I-frame
	// ffmpeg -i input.mp4 -vf "select='eq(pict_type\,I)'" -vsync vfr -frame_pts true outputDir/out_%04d.jpg

	// 构建 FFmpeg 命令参数
	args := []string{
		"-i", inputVideo, // 输入视频文件
		"-vf", "select='eq(pict_type\\,I)'", // 选择 I 帧
		"-vsync", "vfr", // 可变帧率
		"-frame_pts", "true", // 使用帧的时间戳作为文件名的一部分
		"-y", // 覆盖输出文件
		fmt.Sprintf("%s/out_%%04d.jpg", outputDir), // 输出图片文件路径
	}
	// 执行 FFmpeg 命令
	// cmd := exec.Command("ffmpeg", "-i", inputVideo, "-vf", "select='eq(pict_type\\,I)'", "-vsync", "vfr", "-frame_pts", "true", "-y", fmt.Sprintf("%s/out_%%04d.jpg", outputDir))
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	// log print ffmpeg command
	cmdDbg := "ffmpeg"
	for _, arg := range args {
		cmdDbg += fmt.Sprintf(" %s", arg)
	}
	log.Info("Executing command:", cmdDbg)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg execution failed: %v", err)
	}

	return nil
}
