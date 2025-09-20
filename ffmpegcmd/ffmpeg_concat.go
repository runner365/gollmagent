package ffmpegcmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	log "github.com/gollmagent/logging"
)

// ConcatVideosWithResize 合并多个MP4文件，统一缩放至目标分辨率（默认1920x1080）
func ConcatVideosWithResize(files []string, outputFile string) error {
	if len(files) == 0 {
		return fmt.Errorf("no input files provided")
	}

	// 构造FFmpeg命令
	var args []string

	// 添加输入文件
	for _, file := range files {
		args = append(args, "-i", file)
	}

	// 构造filter_complex
	filter := buildFilterComplex(files)
	args = append(args, "-filter_complex", filter)
	args = append(args, "-map", "[outv]", "-map", "[outa]") // 映射输出流
	args = append(args, "-c:v", "libx264", "-crf", "23")    // 可选：设置编码参数
	args = append(args, "-preset", "fast")                  // 可选：编码速度
	args = append(args, outputFile)                         // 输出文件

	// 打印完整命令（调试用）
	log.Info("FFmpeg command:", "ffmpeg", strings.Join(args, " "))

	// 执行FFmpeg
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg execution failed: %v", err)
	}

	return nil
}

// buildFilterComplex 构造FFmpeg的filter_complex参数
func buildFilterComplex(files []string) string {
	var filters []string
	targetWidth, targetHeight := 1920, 1080 // 目标分辨率（可调整）

	// 对每个视频进行缩放和填充
	for i := range files {
		scaleFilter := fmt.Sprintf(
			"[%d:v]scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2,setsar=1[v%d]",
			i, targetWidth, targetHeight, targetWidth, targetHeight, i,
		)
		filters = append(filters, scaleFilter)
	}

	// 构造concat部分
	var concatInputs []string
	for i := range files {
		concatInputs = append(concatInputs, fmt.Sprintf("[v%d][%d:a]", i, i))
	}
	concatFilter := fmt.Sprintf("%sconcat=n=%d:v=1:a=1[outv][outa]", strings.Join(concatInputs, ""), len(files))

	// 合并所有filter
	fullFilter := strings.Join(filters, "; ") + ";" + concatFilter
	return fullFilter
}
