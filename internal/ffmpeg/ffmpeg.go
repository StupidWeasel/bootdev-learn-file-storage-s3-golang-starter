package ffmpeg

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func ProcessVideoForFastStart(filePath string) (string, error) {

	target, err := os.Stat(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", errors.New("Error, that file does not exist")
		}
		return "", errors.New("Error, unable to stat that filepath")
	}

	if target.IsDir() {
		return "", errors.New("Error, that is a directory, expecting a file")
	}

	processingFile := fmt.Sprintf("%s.processing", filePath)

	cmd := exec.Command("ffmpeg",
		"-i", filePath,
		"-c", "copy",
		"-movflags", "faststart",
		"-f", "mp4", processingFile,
	)

	err = cmd.Run()
	if err != nil {
		os.Remove(processingFile)
		return "", fmt.Errorf("Error running ffmpeg command: %v", err)
	}

	ext := filepath.Ext(filePath)
	base := filePath[:len(filePath)-len(ext)]
	newPath := fmt.Sprintf("%s.faststart%s", base, ext)

	err = os.Rename(processingFile, newPath)
	if err != nil {
		os.Remove(processingFile)
		return "", fmt.Errorf("Error renaming file: %v", err)
	}

	return newPath, nil
}
