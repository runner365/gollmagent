package ffmpegcmd

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	log "github.com/gollmagent/logging"
	"github.com/gollmagent/pub"
)

var timeRe = regexp.MustCompile(`time=(\d{2}):(\d{2}):(\d{2}\.\d{2})`)

var VideoResolutions = map[string]int{
	"360p":  360,
	"480p":  480,
	"720p":  720,
	"1080p": 1080,
}

func GetVideoResolution(reslution string, inputW int, inputH int) (w int, h int, err error) {
	var ratio float64
	var value int
	var ok bool

	if inputW <= 0 || inputH <= 0 {
		err = fmt.Errorf("invalid input dimensions: %dx%d", inputW, inputH)
		return
	}
	ratio = float64(inputW) / float64(inputH)

	value, ok = VideoResolutions[reslution]
	if !ok {
		err = fmt.Errorf("unsupported resolution: %s", reslution)
		return
	}
	if inputW > inputH {
		w = int(float64(value) * ratio)
		h = value
	} else {
		w = value
		h = int(float64(value) / ratio)
	}

	w = (w + 1) / 2 * 2
	h = (h + 1) / 2 * 2
	return
}

// 把 "hh:mm:ss.ss" 转成秒
func parseTime(ts string) float64 {
	var h, m, s float64
	fmt.Sscanf(ts, "%f:%f:%f", &h, &m, &s)
	return h*3600 + m*60 + s
}

func TranscodeWithProgress(id string, in string, w int, h int, out string, duration float64, progressObj pub.ProgressCallback) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	args := []string{
		"-i", in,
		"-c:v", "libx264",
		"-vf", fmt.Sprintf("scale=%d:%d", w, h),
		"-r", "30", "-g", "90",
		"-c:a", "aac", "-ar", "48000", "-ac", "2", "-ab", "64k",
		"-f", "mp4", "-y", out,
	}
	argsStr := strings.Join(args, " ")
	log.Infof("Starting ffmpeg with args: %s", argsStr)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Errorf("无法获取ffmpeg stderr: %v", err)
		progressObj.OnProgress(&pub.ProgressInfo{
			Progress: 0,
			Message:  "无法获取ffmpeg stderr, 转码失败",
			Done:     true,
		}, id)
		return err
	}
	log.Infof("ffmpeg command start: ffmpeg %s", argsStr)
	if err := cmd.Start(); err != nil {
		log.Errorf("无法启动ffmpeg: %v", err)
		progressObj.OnProgress(&pub.ProgressInfo{
			Progress: 0,
			Message:  "无法启动ffmpeg, 转码失败",
			Done:     true,
		}, id)
		return err
	}
	log.Infof("ffmpeg command started: ffmpeg %s", argsStr)

	progressObj.OnProgress(&pub.ProgressInfo{
		Progress: 0,
		Message:  "转码开始",
		Done:     false,
	}, id)
	reader := bufio.NewReader(stderr)
	go func(cb pub.ProgressCallback, callId string) {
		log.Infof("开始监控转码进度, callId: %s", callId)
		startMs := time.Now().UnixMilli()
		for {
			line, err := reader.ReadString('\r')
			if err != nil {
				break
			}
			// log.Infof("scanner line: %s", line)
			m := timeRe.FindStringSubmatch(line)
			if len(m) > 0 {
				cur := parseTime(m[1] + ":" + m[2] + ":" + m[3])
				pct := cur / duration * 100
				log.Infof("进度: %.1f%%  当前: %.1fs / %.1fs", pct, cur, duration)

				cb.OnProgress(&pub.ProgressInfo{
					Progress: float32(pct / 100),
					Message:  fmt.Sprintf("转码中 %.1f%%, 耗时(ms): %d", pct, time.Now().UnixMilli()-startMs),
					Done:     false,
				}, callId)
			}
		}
	}(progressObj, id)

	err = cmd.Wait()
	if err != nil {
		log.Errorf("ffmpeg 进程出错: %v", err)
		progressObj.OnProgress(&pub.ProgressInfo{
			Progress: 0,
			Message:  fmt.Sprintf("ffmpeg 进程出错: %v", err),
			Done:     true,
		}, id)
		return err
	}
	progressObj.OnProgress(&pub.ProgressInfo{
		Progress: 1.0,
		Message:  "转码完成",
		Done:     true,
	}, id)
	log.Infof("ffmpeg command finished successfully: ffmpeg %s", argsStr)
	return nil
}
