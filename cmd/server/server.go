package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zheki1/yaprmtrc/internal/repository"
)

// Server — центральная структура HTTP-сервера сбора метрик.
// Содержит хранилище, логгер, файловое хранилище, подключение к БД и настройки аудита.
type Server struct {
	storage       repository.Repository
	logger        Logger
	fileStorage   *FileStorage
	syncSave      bool
	db            *pgxpool.Pool
	key           string
	audit         *AuditPublisher
	cryptoKey     string
	trustedSubnet string
}

func (s *Server) saveIfNeeded() {
	if s.syncSave {
		metrics, err := s.storage.GetAll(context.Background())
		if err != nil {
			s.logger.Fatalf(err.Error())
		}
		if err := s.fileStorage.Save(metrics); err != nil {
			s.logger.Fatalf("Sync save failed")
		}
	}
}
