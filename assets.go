package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type AssetFile struct {
	FileName  string
	Path      string
	URL       string
	MimeType  string
	CreatedAt time.Time
	Size      int64
	UserID    uuid.UUID
}

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func (cfg apiConfig) addToAssetsDir(data io.Reader, asset *AssetFile, fn func(AssetFile) error) error {

	path := filepath.Join(cfg.assetsRoot, asset.FileName)
	asset.Path = path
	asset.URL = fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, asset.FileName)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	success := false
	defer func() {
		if !success {
			os.Remove(path)
		}
	}()

	_, err = io.Copy(f, data)
	if err != nil {
		return err
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	if fn != nil {
		if err := fn(*asset); err != nil {
			return err
		}
	}
	success = true
	asset.CreatedAt = fileInfo.ModTime()
	asset.Size = fileInfo.Size()

	return nil
}
