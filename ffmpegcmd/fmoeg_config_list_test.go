package ffmpegcmd

import (
	"fmt"
	"testing"
)

func TestConfigList(t *testing.T) {
	cfg, err := GetFFmpegConfig()
	if err != nil {
		t.Fatalf("GetFFmpegConfig failed: %v", err)
	}
	if len(cfg) == 0 {
		t.Fatal("GetFFmpegConfig returned empty config")
	}
	fmt.Println("ffmpeg config:", cfg)
	t.Logf("FFmpeg Config: %v", cfg)
}
