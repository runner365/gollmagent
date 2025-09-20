package ffprobe

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestGetMediaInfo(t *testing.T) {
	info, err := GetMediaFullInfo("../../input.mp4")
	if err != nil {
		fmt.Println("Error getting media info:", err)
		t.Errorf("Error getting media info: %v", err)
		return
	}
	data, _ := json.Marshal(&info)
	fmt.Println("Media info:", string(data))
	t.Logf("Media info: %+v", info)
}
