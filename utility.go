package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"math"
	"os"

	"github.com/stupidweasel/learn-file-storage-s3-golang-starter/internal/ffmpeg"
)

func generateRandomKey(length int) string {
	randomBytes := make([]byte, length)
	rand.Read(randomBytes)
	return base64.RawURLEncoding.EncodeToString(randomBytes)
}

func aspectRatio(width, height int) float64 {
	return float64(width) / float64(height)
}

func getVideoAspectRatio(filepath string) (string, error) {

	target, err := os.Stat(filepath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", errors.New("Error, that file does not exist")
		}
		return "", errors.New("Error, unable to stat that filepath")
	}

	if target.IsDir() {
		return "", errors.New("Error, that is a directory, expecting a file")
	}

	ffprobeResult, err := ffmpeg.FfprobeVideo(filepath)
	if err != nil {
		return "", err
	}

	streamWidth := ffprobeResult.Streams[0].Width
	streamHeight := ffprobeResult.Streams[0].Height
	ratio := aspectRatio(streamWidth, streamHeight)
	tolerance := 0.05

	switch {
	case math.Abs(ratio-1.778) < tolerance:
		return "landscape", nil
	case math.Abs(ratio-0.5625) < tolerance:
		return "portrait", nil
	default:
		return "other", nil
	}

}
