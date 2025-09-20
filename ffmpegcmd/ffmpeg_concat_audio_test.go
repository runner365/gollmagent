package ffmpegcmd

import (
	"fmt"
	"testing"
)

func TestConcatAudioFiles(t *testing.T) {
	inputFiles := []string{"../1.mp4", "../2.mp4", "../3.mp4"}
	outputFile := "output_concat_audio.m4a"
	err := ConcatAudioOnly(inputFiles, outputFile)
	if err != nil {
		fmt.Println("ConcatAudioOnly failed:", err)
		t.Fatalf("ConcatAudioOnly failed: %v", err)
	}
	fmt.Printf("Merged audio file created: %s\n", outputFile)
}
