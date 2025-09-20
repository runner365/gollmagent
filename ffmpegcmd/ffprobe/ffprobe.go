package ffprobe

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
)

type Stream struct {
	CodecType    string `json:"codec_type"`
	CodecName    string `json:"codec_name"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	SampleRate   string `json:"sample_rate,omitempty"`
	Channels     int    `json:"channels,omitempty"`
	GOPSize      int    `json:"gop_size,omitempty"` // 部分文件存在
	AvgFrameRate string `json:"avg_frame_rate"`     // 形如 "30/1" 或 "30000/1001"
}

type ProbeResp struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
	Streams []Stream `json:"streams"`
}

type FullInfo struct {
	Duration   float64 `json:"duration"` // 秒
	HasVideo   bool    `json:"has_video"`
	HasAudio   bool    `json:"has_audio"`
	VideoCodec string  `json:"video_codec"`
	AudioCodec string  `json:"audio_codec"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	FrameRate  float64 `json:"frame_rate"`
	SampleRate int     `json:"sample_rate"`
	Channels   int     `json:"channels"`
}

func GetMediaFullInfo(filename string) (FullInfo, error) {
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filename,
	}
	out, err := exec.Command("ffprobe", args...).Output()
	if err != nil {
		return FullInfo{}, err
	}
	var pr ProbeResp
	if err := json.Unmarshal(out, &pr); err != nil {
		return FullInfo{}, err
	}

	dur, _ := strconv.ParseFloat(pr.Format.Duration, 64)
	info := FullInfo{Duration: dur}
	for _, s := range pr.Streams {
		switch s.CodecType {
		case "video":
			info.HasVideo = true
			info.VideoCodec = s.CodecName
			info.Width = s.Width
			info.Height = s.Height
			// 解析帧率
			if numden := strings.Split(s.AvgFrameRate, "/"); len(numden) == 2 {
				num, _ := strconv.ParseFloat(numden[0], 64)
				den, _ := strconv.ParseFloat(numden[1], 64)
				if den != 0 {
					info.FrameRate = num / den
				}
			}
		case "audio":
			info.HasAudio = true
			info.AudioCodec = s.CodecName
			if sr, err := strconv.Atoi(s.SampleRate); err == nil {
				info.SampleRate = sr
			}
			info.Channels = s.Channels
		}
	}
	return info, nil
}
