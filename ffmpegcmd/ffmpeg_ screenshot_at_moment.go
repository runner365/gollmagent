package ffmpegcmd

import (
	"fmt"
	"os"
	"os/exec"

	log "github.com/gollmagent/logging"
)

func ScreenshotOnePictureAtMoment(inputFile string, moment string, outputFile string) error {
	//moment format "00:00:10"
	// ffmpeg -ss 00:00:10 -i input.mp4 -vframes 1 -q:v 2 output.jpg
	args := []string{
		"-ss", moment,
		"-i", inputFile,
		"-vframes", "1",
		"-q:v", "2",
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
