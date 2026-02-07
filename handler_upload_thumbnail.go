package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
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

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	// imageData, err := io.ReadAll(file)
	// if err != nil {
	// 	respondWithError(w, http.StatusBadRequest, "Unable to read file", err)
	// 	return
	// }
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Invalid user", nil)
		return
	}
	content, ext, found := strings.Cut(contentType, "/")
	if !found || content != "image" {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", nil)
		return
	}
	thumbFilename := videoID.String() + "." + ext
	thumbPath := filepath.Join(cfg.assetsRoot, thumbFilename)
	thumbFile, err := os.Create(thumbPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to save thumbnail", err)
		return
	}
	size, err := io.Copy(thumbFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to save thumbnail", err)
		return
	}
	log.Printf("Created thumbnail %s %d bytes", thumbPath, size)
	thumbUrl := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, thumbFilename)
	video.ThumbnailURL = &thumbUrl
	if err := cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to save thumbnail", err)
		return
	}
	respondWithJSON(w, http.StatusOK, video)
}
