package ffmpegcmd

import (
	"fmt"
	"os/exec"

	log "github.com/gollmagent/logging"
	"github.com/gollmagent/utils"
)

func MergeM3U8ToMP4(m3u8Path string, outputMp4Path string) error {
	// check if m3u8 file exists
	if !utils.FileExists(m3u8Path) {
		log.Errorf("m3u8 file does not exist: %s", m3u8Path)
		return fmt.Errorf("m3u8 file does not exist: %s", m3u8Path)
	}

	// generate ffmpeg command
	arguments := []string{
		"-i", m3u8Path,
		"-c", "copy",
		"-bsf:a", "aac_adtstoasc",
		outputMp4Path,
	}

	cmdDbg := "ffmpeg "
	for _, arg := range arguments {
		cmdDbg += fmt.Sprintf(" %q", arg)
	}
	log.Infof("Executing command: %s", cmdDbg)

	cmd := exec.Command("ffmpeg", arguments...)
	err := cmd.Run()
	if err != nil {
		log.Errorf("error merging m3u8 to mp4: %v", err)
		return fmt.Errorf("error merging m3u8 to mp4: %v", err)
	}
	log.Infof("Successfully merged m3u8 to mp4: %s", outputMp4Path)
	return nil
}
