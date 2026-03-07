package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/video"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	// used for if someone try upload more than 1GB then it will stop reading and return error
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)

	defer r.Body.Close()

	// getting the video id based on the path value of upload endpoint
	videoIDString := r.PathValue("videoID")

	// get the uuid of the video id string
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	// authorization
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

	// get the video metadata saved when user created in tubely
	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video metadata", err)
		return
	}
	if videoMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You don't own this video", nil)
		return
	}

	// extract uploaded video from http request when client submit for with input name "video"
	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	//get the extension file of the uploaded file
	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid video format", nil)
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create temp file", err)
		return
	}

	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	//  Copies the uploaded video bytes from the in-memory stream (file) into the temp file on disk. After this, tempFile contains the full video.
	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't write video file", err)
		return
	}

	// moves cursor that at the end of the file into start of the file
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't write video file", err)
		return
	}

	parts := strings.Split(mediaType, "/")
	ext := parts[len(parts)-1]

	randBytes := make([]byte, 32)
	_, err = rand.Read(randBytes)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't generate random file name", err)
		return
	}

	processedFilePath, err := video.ProcessVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't process video", err)
		return
	}
	defer os.Remove(processedFilePath)

	processedFile, err := os.Open(processedFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't open processed video file", err)
		return
	}
	defer processedFile.Close()

	aspectRatio, err := video.GetVideoAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video aspect ratio", err)
		return
	}
	log.Printf("Video aspect ratio: %s", aspectRatio)

	var screenType string
	switch aspectRatio {
	case "16:9":
		screenType = "landscape"
	case "9:16":
		screenType = "portrait"
	default:
		screenType = "square"
	}

	encodedFileName := base64.RawURLEncoding.EncodeToString(randBytes)
	fullFileName := fmt.Sprintf("%s/%s.%s", screenType, encodedFileName, ext)

	s3InputParam := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &fullFileName,
		Body:        processedFile,
		ContentType: &mediaType,
	}
	_, err = cfg.s3Client.PutObject(context.Background(), &s3InputParam)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't upload video", err)
		return
	}

	videoUrl := fmt.Sprintf("%s/%s", cfg.s3CfDistribution, fullFileName)
	videoMetadata.VideoURL = &videoUrl

	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video metadata", err)
		return
	}
}
