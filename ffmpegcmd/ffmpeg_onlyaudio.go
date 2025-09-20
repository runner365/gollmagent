package ffmpegcmd

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	log "github.com/gollmagent/logging"
)

func GetM4aFromMediaFile(inputFile string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	outputFile := strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + "_audio.m4a"

	args := []string{"-y", "-i", inputFile,
		"-vn",
		"-acodec", "aac",
		"-ac", "2",
		"-ar", "44100",
		"-b:a", "64k", outputFile}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	cmdDbg := "ffmpeg"
	for _, a := range args {
		cmdDbg += " " + a
	}
	log.Infof("Running command: %s", cmdDbg)

	err := cmd.Run()
	if err != nil {
		log.Errorf("ffmpeg command failed: %v, output: %s", err, out.String())
		return "", err
	}
	log.Infof("ffmpeg command succeeded, output file: %s", outputFile)
	return outputFile, nil
}
