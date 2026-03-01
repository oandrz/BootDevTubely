package video

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

func GetVideoAspectRatio(filePath string) (string, error) {
	var buf bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	cmd.Stdout = &buf
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("ffprobe error: %s: %w", stderr.String(), err)
	}

	videosMetaData := struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}{}
	err = json.Unmarshal(buf.Bytes(), &videosMetaData)
	if err != nil {
		return "", err
	}

	if len(videosMetaData.Streams) == 0 {
		return "", fmt.Errorf("no video streams found in file: %s", filePath)
	}

	w := videosMetaData.Streams[0].Width
	h := videosMetaData.Streams[0].Height
	if h == 0 {
		return "", fmt.Errorf("invalid video height: 0")
	}

	return getAspectRatioLabel(w, h), nil
}

func getAspectRatioLabel(width, height int) string {
	ratio := float64(width) / float64(height)
	if ratio > 1 {
		return "16:9"
	} else if ratio < 1 {
		return "9:16"
	}
	return "1:1"
}
