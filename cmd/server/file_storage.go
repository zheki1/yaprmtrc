package main

import (
	"encoding/json"
	"os"

	"github.com/zheki1/yaprmtrc/internal/models"
)

// FileStorage обеспечивает сохранение и восстановление метрик в JSON-файл для переживания перезапусков.
type FileStorage struct {
	path string
}

// NewFileStorage создаёт FileStorage с указанным путём к файлу.
func NewFileStorage(path string) *FileStorage {
	return &FileStorage{path: path}
}

// Save сериализует срез метрик в JSON и записывает в файл.
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

// Load читает метрики из JSON-файла.
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
