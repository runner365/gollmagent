package ffmpegcmd

import (
	"fmt"
	"os"
	"os/exec"

	log "github.com/gollmagent/logging"
)

func Srt2Video(inputFile, srtFile, outputFile string) error {
	// ffmpeg -i input.mp4 -i subtitles.srt -c:a copy -c:s mov_text -f mp4 -y output.mp4
	args := []string{
		"-i", inputFile,
		"-i", srtFile,
		"-c:v", "copy",
		"-c:a", "copy",
		"-c:s", "mov_text",
		"-f", "mp4",
		"-y", outputFile,
	}

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

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
