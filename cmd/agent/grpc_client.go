package main

import (
	"context"
	"fmt"

	"github.com/zheki1/yaprmtrc/internal/models"
	"github.com/zheki1/yaprmtrc/internal/proto/pb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// GRPCClient — клиент для отправки метрик через gRPC.
type GRPCClient struct {
	conn    *grpc.ClientConn
	client  pb.MetricsClient
	logger  *zap.SugaredLogger
	agentIP string
}

// NewGRPCClient создаёт новый gRPC клиент.
func NewGRPCClient(addr string, agentIP string, logger *zap.SugaredLogger) (*GRPCClient, error) {
	conn, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	return &GRPCClient{
		conn:    conn,
		client:  pb.NewMetricsClient(conn),
		logger:  logger,
		agentIP: agentIP,
	}, nil
}

// SendMetrics отправляет батч метрик на сервер через gRPC.
func (gc *GRPCClient) SendMetrics(ctx context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	// Преобразуем метрики из модели в proto-формат
	pbMetrics := make([]*pb.Metric, 0, len(metrics))
	for _, m := range metrics {
		pbMetric := &pb.Metric{
			Id:   m.ID,
			Type: modelTypeToPBType(m.MType),
		}

		switch m.MType {
		case models.Counter:
			if m.Delta != nil {
				pbMetric.Delta = *m.Delta
			}
		case models.Gauge:
			if m.Value != nil {
				pbMetric.Value = *m.Value
			}
		}

		pbMetrics = append(pbMetrics, pbMetric)
	}

	// Создаём контекст с метаданными (IP-адрес агента)
	ctx = metadata.AppendToOutgoingContext(ctx, "x-real-ip", gc.agentIP)

	// Отправляем метрики на сервер
	req := &pb.UpdateMetricsRequest{
		Metrics: pbMetrics,
	}

	_, err := gc.client.UpdateMetrics(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update metrics: %w", err)
	}

	gc.logger.Debugf("sent %d metrics via gRPC", len(metrics))
	return nil
}

// Close закрывает соединение с gRPC сервером.
func (gc *GRPCClient) Close() error {
	if gc.conn != nil {
		return gc.conn.Close()
	}
	return nil
}

// modelTypeToPBType преобразует тип метрики из модели в proto-формат.
func modelTypeToPBType(modelType string) pb.Metric_MType {
	switch modelType {
	case models.Counter:
		return pb.Metric_COUNTER
	case models.Gauge:
		return pb.Metric_GAUGE
	default:
		return pb.Metric_GAUGE
	}
}
