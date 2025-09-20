package ffmpegcmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/gollmagent/ffmpegcmd/ffprobe"
	"github.com/gollmagent/pub"
)

// Mock ProgressCallback
type mockProgressCallback struct {
	updates []pub.ProgressInfo
}

func (m *mockProgressCallback) OnProgress(info *pub.ProgressInfo, id string) {
	fmt.Println("Progress:", info.Progress, "Message:", info.Message, "Done:", info.Done)
	m.updates = append(m.updates, *info)
}

func (m *mockProgressCallback) CheckProgress(id string) *pub.ProgressInfo {
	return &pub.ProgressInfo{
		Progress: 1.0,
		Message:  "Transcoding complete",
		Done:     true,
	}
}

type InputVideoInfo struct {
	SrcWidth  int
	SrcHeight int
	Res       string
}

func TestGetVideoResolution(t *testing.T) {
	infos := []InputVideoInfo{
		{1920, 1080, "1080p"},
		{1920, 1080, "720p"},
		{1920, 1080, "480p"},
		{1920, 1080, "360p"},
		{1080, 1920, "1080p"},
		{1080, 1920, "720p"},
		{1080, 1920, "480p"},
		{1080, 1920, "360p"},
	}
	for _, info := range infos {
		w, h, err := GetVideoResolution(info.Res, info.SrcWidth, info.SrcHeight)
		if err != nil {
			t.Errorf("GetVideoResolution failed for %v: %v", info, err)
			continue
		}
		fmt.Printf("Input: %dx%d, Res: %s => Output: %dx%d\n", info.SrcWidth, info.SrcHeight, info.Res, w, h)
		t.Logf("Input: %dx%d, Res: %s => Output: %dx%d", info.SrcWidth, info.SrcHeight, info.Res, w, h)
	}
}

func TestTranscodeWithProgress(t *testing.T) {
	vRes := "480p"
	// 准备输入输出文件路径
	inFile := "/Users/wei.shi/Documents/movies/weibo4min_h265_opus.mp4" // 请确保有此测试文件
	outFile := "test_out.mp4"
	defer os.Remove(outFile)

	info, err := ffprobe.GetMediaFullInfo(inFile)
	if err != nil {
		t.Fatalf("GetMediaFullInfo failed: %v", err)
	}
	fmt.Printf("Media info: %+v\n", info)

	w, h, err := GetVideoResolution(vRes, info.Width, info.Height)
	if err != nil {
		fmt.Printf("GetVideoResolution failed: %v\n", err)
		t.Fatalf("GetVideoResolution failed: %v", err)
		return
	}
	cb := &mockProgressCallback{}
	err = TranscodeWithProgress("testid", inFile, w, h, outFile, info.Duration, cb)
	if err != nil {
		t.Fatalf("TranscodeWithProgress failed: %v", err)
	}

	// 检查输出文件是否生成
	if _, err := os.Stat(outFile); err != nil {
		t.Errorf("Output file not created: %v", err)
	}

	// 检查进度回调是否被调用
	if len(cb.updates) == 0 {
		t.Errorf("No progress updates received")
	}
}
