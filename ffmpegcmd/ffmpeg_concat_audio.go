package ffmpegcmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ConcatAudioOnly 提取多个媒体文件的音频并合并为 M4A
func ConcatAudioOnly(inputFiles []string, outputFile string) error {
	if len(inputFiles) == 0 {
		return fmt.Errorf("no input files provided")
	}

	// 构造 FFmpeg 命令
	args := []string{}
	for _, file := range inputFiles {
		args = append(args, "-i", file)
	}

	// 构造 filter_complex（仅合并音频）
	filter := buildAudioFilter(inputFiles)
	args = append(args, "-filter_complex", filter)
	args = append(args, "-map", "[outa]") // 映射音频流
	args = append(args, "-c:a", "aac")    // AAC 编码
	args = append(args, "-b:a", "64k")    // 比特率（可选）
	args = append(args, "-ar", "44100")   // 音频采样率
	args = append(args, "-ac", "2")       // 双声道
	args = append(args, "-vn")            // 禁用视频流
	args = append(args, "-y", outputFile) // 输出文件（-y 覆盖已存在文件）

	// 打印命令（调试用）
	fmt.Println("FFmpeg command:", "ffmpeg", strings.Join(args, " "))

	// 执行命令
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg execution failed: %v", err)
	}

	return nil
}

// buildAudioFilter 构造音频合并的 filter_complex
func buildAudioFilter(files []string) string {
	var inputs []string
	for i := range files {
		inputs = append(inputs, fmt.Sprintf("[%d:a]", i)) // 提取每个文件的音频流
	}
	filter := fmt.Sprintf("%sconcat=n=%d:v=0:a=1[outa]", strings.Join(inputs, ""), len(files))
	return filter
}
