package main

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/zheki1/yaprmtrc.git/internal/repository"
)

type Server struct {
	storage     repository.Repository
	logger      Logger
	fileStorage *FileStorage
	syncSave    bool
	db          *pgx.Conn
}

func (s *Server) saveIfNeeded() error {
	if s.syncSave {
		metrics, err := s.storage.GetAll(context.Background())
		if err != nil {
			s.logger.Fatalf(err.Error())
			return err
		}
		if err := s.fileStorage.Save(metrics); err != nil {
			s.logger.Fatalf("Sync save failed")
			return err
		}
	}

	return nil
}
