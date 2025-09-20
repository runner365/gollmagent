package ffmpegcmd

import (
	"fmt"
	"regexp"
)

func IsValidFFmpegTimeFormat(timeStr string) bool {
	// 简单的正则表达式来匹配 HH:MM:SS 格式
	var timeFormatRe = `^(\d{1,2}):([0-5]?\d):([0-5]?\d)(\.\d+)?$`
	matched, _ := regexp.MatchString(timeFormatRe, timeStr)
	return matched
}

func GetTimeFromTimeFormat(timeStr string) (hours, minutes, seconds int, err error) {
	var timeFormatRe = regexp.MustCompile(`^(\d{1,2}):([0-5]?\d):([0-5]?\d)(\.\d+)?$`)
	matches := timeFormatRe.FindStringSubmatch(timeStr)
	if len(matches) < 4 {
		return 0, 0, 0, nil // 或者返回一个错误，表示格式不正确
	}

	fmt.Sscanf(matches[1], "%d", &hours)
	fmt.Sscanf(matches[2], "%d", &minutes)
	fmt.Sscanf(matches[3], "%d", &seconds)

	return hours, minutes, seconds, nil
}
