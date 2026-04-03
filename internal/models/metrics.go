// Package models определяет структуры данных для обмена метриками между агентом и сервером.
package models

const (
	// Counter — тип метрики "счётчик" (кумулятивное целочисленное значение).
	Counter = "counter"
	// Gauge — тип метрики "измерение" (произвольное значение с плавающей точкой).
	Gauge = "gauge"
)

// Metrics описывает одну метрику, передаваемую через JSON API.
// Поле MType принимает значение [Counter] или [Gauge].
// Для counter используется поле Delta, для gauge — поле Value.
type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}
