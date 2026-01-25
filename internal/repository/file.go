package repository

import (
	"context"
	"encoding/json"
	"os"
	"sync"

	"github.com/zheki1/yaprmtrc.git/internal/models"
)

type FileRepository struct {
	path string
	mu   sync.Mutex
}

func NewFileRepository(path string) *FileRepository {
	return &FileRepository{path: path}
}

func (f *FileRepository) save(metrics []models.Metrics) error {
	tmp := f.path + ".tmp"

	file, err := os.Create(tmp)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")

	if err := enc.Encode(metrics); err != nil {
		file.Close()
		return err
	}

	if err := file.Close(); err != nil {
		return err
	}

	return os.Rename(tmp, f.path)
}

func (f *FileRepository) restore() ([]models.Metrics, error) {
	file, err := os.Open(f.path)
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

func (f *FileRepository) UpdateGauge(
	ctx context.Context,
	name string,
	value float64,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	metrics, err := f.restore()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	updated := false

	for i := range metrics {
		if metrics[i].ID == name && metrics[i].MType == models.Gauge {
			metrics[i].Value = &value
			updated = true
			break
		}
	}

	if !updated {
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &value,
		})
	}

	return f.save(metrics)
}

func (f *FileRepository) UpdateCounter(
	ctx context.Context,
	name string,
	delta int64,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	metrics, err := f.restore()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	for i := range metrics {
		if metrics[i].ID == name && metrics[i].MType == models.Counter {
			*metrics[i].Delta += delta
			return f.save(metrics)
		}
	}

	metrics = append(metrics, models.Metrics{
		ID:    name,
		MType: models.Counter,
		Delta: &delta,
	})

	return f.save(metrics)
}

func (f *FileRepository) GetAll(
	ctx context.Context,
) ([]models.Metrics, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.restore()
}

func (f *FileRepository) GetGauge(
	ctx context.Context,
	name string,
) (float64, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	metrics, err := f.restore()
	if err != nil {
		return 0, false, err
	}

	for _, m := range metrics {
		if m.ID == name && m.MType == models.Gauge {
			return *m.Value, true, nil
		}
	}

	return 0, false, nil
}

func (f *FileRepository) GetCounter(
	ctx context.Context,
	name string,
) (int64, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	metrics, err := f.restore()
	if err != nil {
		return 0, false, err
	}

	for _, m := range metrics {
		if m.ID == name && m.MType == models.Counter {
			return *m.Delta, true, nil
		}
	}

	return 0, false, nil
}

func (f *FileRepository) UpdateBatch(
	ctx context.Context,
	metrics []models.Metrics,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := f.restore()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	for _, m := range metrics {
		updated := false

		for i := range data {
			if data[i].ID == m.ID && data[i].MType == m.MType {

				if m.MType == models.Gauge {
					data[i].Value = m.Value
				}

				if m.MType == models.Counter {
					*data[i].Delta += *m.Delta
				}

				updated = true
			}
		}

		if !updated {
			data = append(data, m)
		}
	}

	return f.save(data)
}

func (f *FileRepository) Close() error {
	return nil
}
