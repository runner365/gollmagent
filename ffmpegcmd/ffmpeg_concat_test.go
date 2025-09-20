package ffmpegcmd

import (
	"fmt"
	"testing"
)

func TestConcat(t *testing.T) {
	inputFiles := []string{"../1.mp4", "../2.mp4", "../3.mp4"}
	outputFile := "output_concat.mp4"
	err := ConcatVideosWithResize(inputFiles, outputFile)
	if err != nil {
		fmt.Println("ConcatVideosWithResize failed:", err)
		t.Fatalf("ConcatVideosWithResize failed: %v", err)
	}
	fmt.Printf("Merged file created: %s\n", outputFile)
}
