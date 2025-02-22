package main

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/stupidweasel/learn-file-storage-s3-golang-starter/internal/auth"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)
	r.Body = http.MaxBytesReader(w, r.Body, MaxMemory)

	if err := r.ParseMultipartForm(MaxMemory); err != nil {
		respondWithError(w, http.StatusRequestEntityTooLarge, "Request body too large", err)
		return
	}

	file, fileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			respondWithError(w, http.StatusBadRequest, "Missing required 'thumbnail' file in form-data", err)
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

	videoDetails, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to fetch video details", err)
		return
	}

	if videoDetails.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Video does not belong to this user", err)
		return
	}

	fileType, _, err := mime.ParseMediaType(fileHeaderContentType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to get media type", err)
		return
	}

	ext, ok := cfg.allowedThumbnailMimes[fileType]
	if !ok {
		respondWithError(w, http.StatusUnsupportedMediaType, "Invalid media type", err)
		return
	}

	thumbName := generateRandomKey(32)

	asset := AssetFile{
		FileName: fmt.Sprintf("%s.%s", thumbName, ext),
		MimeType: fileHeaderContentType,
		UserID:   userID,
	}

	dbFunc := func(asset AssetFile) error {
		videoDetails.ThumbnailURL = &asset.URL
		err = cfg.db.UpdateVideo(videoDetails)
		if err != nil {
			return err
		}
		return nil
	}

	err = cfg.addToAssetsDir(file, &asset, dbFunc)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to read file", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoDetails)
}
