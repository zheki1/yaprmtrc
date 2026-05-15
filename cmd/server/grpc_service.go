package main

import (
	"context"
	"fmt"

	"github.com/zheki1/yaprmtrc/internal/models"
	"github.com/zheki1/yaprmtrc/internal/proto/pb"
)

// MetricsServiceImpl реализует gRPC сервис Metrics.
type MetricsServiceImpl struct {
	pb.UnimplementedMetricsServer
	server *Server
}

// NewMetricsServiceImpl создаёт новый gRPC сервис для работы с метриками.
func NewMetricsServiceImpl(s *Server) *MetricsServiceImpl {
	return &MetricsServiceImpl{
		server: s,
	}
}

// UpdateMetrics реализует метод UpdateMetrics сервиса Metrics.
// Принимает батч метрик, преобразует их из proto-формата в модель и сохраняет.
func (m *MetricsServiceImpl) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	if req == nil || len(req.Metrics) == 0 {
		return &pb.UpdateMetricsResponse{}, nil
	}

	for _, metric := range req.GetMetrics() {
		if metric == nil {
			continue
		}

		// Преобразуем proto-метрику в модель Metrics
		modelMetric := &models.Metrics{
			ID:    metric.GetId(),
			MType: pbTypeToModel(metric.GetType()),
		}

		// В зависимости от типа метрики устанавливаем значение
		switch metric.GetType() {
		case pb.Metric_COUNTER:
			delta := metric.GetDelta()
			modelMetric.Delta = &delta
		case pb.Metric_GAUGE:
			value := metric.GetValue()
			modelMetric.Value = &value
		default:
			continue
		}

		// Сохраняем метрику
		switch modelMetric.MType {
		case models.Counter:
			if modelMetric.Delta != nil {
				if err := m.server.storage.UpdateCounter(ctx, modelMetric.ID, *modelMetric.Delta); err != nil {
					m.server.logger.Error("failed to save counter metric", fmt.Sprintf("%s: %v", modelMetric.ID, err))
				}
			}
		case models.Gauge:
			if modelMetric.Value != nil {
				if err := m.server.storage.UpdateGauge(ctx, modelMetric.ID, *modelMetric.Value); err != nil {
					m.server.logger.Error("failed to save gauge metric", fmt.Sprintf("%s: %v", modelMetric.ID, err))
				}
			}
		}
	}

	// Синхронное сохранение в файл, если требуется
	m.server.saveIfNeeded()

	return &pb.UpdateMetricsResponse{}, nil
}

// pbTypeToModel преобразует тип метрики из proto-формата в модель.
func pbTypeToModel(pbType pb.Metric_MType) string {
	switch pbType {
	case pb.Metric_COUNTER:
		return models.Counter
	case pb.Metric_GAUGE:
		return models.Gauge
	default:
		return models.Gauge
	}
}

// modelTypeToPB преобразует тип метрики из модели в proto-формат.
func modelTypeToPB(modelType string) pb.Metric_MType {
	switch modelType {
	case models.Counter:
		return pb.Metric_COUNTER
	case models.Gauge:
		return pb.Metric_GAUGE
	default:
		return pb.Metric_GAUGE
	}
}
