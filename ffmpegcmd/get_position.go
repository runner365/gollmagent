package ffmpegcmd

import "fmt"

const pad = 8

func GetPosition(position string, videoX int, videoY int, imageX int, imageY int) (int, int, error) {

	switch position {
	case "top-left":
		return pad, pad, nil
	case "top-right":
		return videoX - imageX - pad, pad, nil
	case "bottom-left":
		return pad, videoY - imageY - pad, nil
	case "bottom-right":
		return videoX - imageX - pad, videoY - imageY - pad, nil
	default:
		return 0, 0, fmt.Errorf("invalid position: %s", position)
	}
}

func GetTextPosition(position string, videoX int, videoY int, fontSize int) (int, int, error) {
	switch position {
	case "top-left":
		return pad, pad, nil
	case "top-right":
		return videoX - fontSize - pad, pad, nil
	case "bottom-left":
		return pad, videoY - fontSize - pad, nil
	case "bottom-right":
		return videoX - fontSize - pad, videoY - fontSize - pad, nil
	default:
		return 0, 0, fmt.Errorf("invalid position: %s", position)
	}
}
