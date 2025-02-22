package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stupidweasel/learn-file-storage-s3-golang-starter/internal/database"
)

const MaxMemory = 10 << 20

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {

	params := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	presignClient := s3.NewPresignClient(s3Client)
	presignedGetObject, err := presignClient.PresignGetObject(context.Background(), params, s3.WithPresignExpires(expireTime))

	if err != nil {
		return "", fmt.Errorf("unable to PresignGetObject: %v", err)
	}

	return presignedGetObject.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {

	if video.VideoURL == nil {
		return video, nil
	}

	videoURL := strings.Split(*video.VideoURL, ",")
	if len(videoURL) != 2 {
		return database.Video{}, errors.New("unexpected VideoURL format")
	}

	presignedURL, err := generatePresignedURL(cfg.s3Client, videoURL[0], videoURL[1], cfg.presignedExpireTime)
	if err != nil {
		return database.Video{}, err
	}

	video.VideoURL = &presignedURL
	return video, nil
}
