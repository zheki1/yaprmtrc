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

func (fs *FileStorage) Save(metrics []models.Metrics) error {
	file, err := os.Create(fs.path)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(metrics)
}

func (fs *FileStorage) Load() ([]models.Metrics, error) {
	file, err := os.Open(fs.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var metrics []models.Metrics
	if err := json.NewDecoder(file).Decode(&metrics); err != nil {
		return nil, err
	}

	return metrics, nil
}
