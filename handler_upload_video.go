package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/uuid"
	"github.com/stupidweasel/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/stupidweasel/learn-file-storage-s3-golang-starter/internal/ffmpeg"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	videoDetails, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to fetch video details", err)
		return
	}

	if videoDetails.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Video does not belong to you", err)
		return
	}

	fmt.Println("uploading video", videoID, "by user", userID)
	r.Body = http.MaxBytesReader(w, r.Body, VideoMaxUploadSize)

	if err := r.ParseMultipartForm(VideoMaxUploadSize); err != nil {
		respondWithError(w, http.StatusRequestEntityTooLarge, "Request body too large", err)
		return
	}

	file, fileHeader, err := r.FormFile("video")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			respondWithError(w, http.StatusBadRequest, "Missing required 'video' file in form-data", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Could not parse request", err)
		return
	}
	defer file.Close()

	fileHeaderContentType := fileHeader.Header.Get("Content-Type")
	if len(fileHeaderContentType) == 0 {
		respondWithError(w, http.StatusBadRequest, "Empty content-type header", nil)
		return
	}

	contentType := strings.Split(fileHeaderContentType, ";")
	if len(contentType) != 1 {
		respondWithError(w, http.StatusBadRequest, "Invalid or malformed content-type header", nil)
		return
	}
	fileHeaderContentType = contentType[0]

	fileType, _, err := mime.ParseMediaType(fileHeaderContentType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to get media type", err)
		return
	}

	ext, ok := cfg.allowedVideoMimes[fileType]
	if !ok {
		respondWithError(w, http.StatusUnsupportedMediaType, "Invalid media type", err)
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-temp-video")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not allocate temp file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not write temp file", err)
		return
	}

	prefix, err := getVideoAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to get video aspect ratio", err)
		return
	}

	thisKey := fmt.Sprintf("%s/%s.%s", prefix, generateRandomKey(32), ext)

	fastStartPath, err := ffmpeg.ProcessVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create faststart version of video", err)
		return
	}

	fastStartFile, err := os.Open(fastStartPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not open faststart video", err)
		return
	}
	defer os.Remove(fastStartFile.Name())
	defer fastStartFile.Close()
	tempFile.Close()

	params := s3.PutObjectInput{
		Bucket:      aws.String(cfg.s3Bucket),
		Key:         aws.String(thisKey),
		Body:        fastStartFile,
		ContentType: aws.String(fileType),
	}

	_, err = cfg.s3Client.PutObject(context.Background(), &params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to PutObject", err)
		return
	}

	newVideoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, thisKey)
	videoDetails.VideoURL = &newVideoURL
	err = cfg.db.UpdateVideo(videoDetails)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoDetails)
}
