package main

import (
	"encoding/json"
	"os"

	"github.com/zheki1/yaprmtrc.git/internal/models"
)

type FileStorage struct {
	path string
}

func NewFileStorage(path string) *FileStorage {
	return &FileStorage{path: path}
}

func (fs *FileStorage) Save(metrics []models.Metrics) (err error) {
	file, err := os.Create(fs.path)
	if err != nil {
		return err
	}

	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")

	if err = enc.Encode(metrics); err != nil {
		return err
	}

	return nil
}

func (fs *FileStorage) Load() (metrics []models.Metrics, err error) {
	file, err := os.Open(fs.path)
	if err != nil {
		return nil, err
	}

	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if err := json.NewDecoder(file).Decode(&metrics); err != nil {
		return nil, err
	}

	return metrics, nil
}
