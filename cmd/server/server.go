package main

import (
	"github.com/jackc/pgx/v5"
)

type Server struct {
	storage     Storage
	logger      Logger
	fileStorage *FileStorage
	syncSave    bool
	db          *pgx.Conn
}

func (s *Server) saveIfNeeded() {
	if s.syncSave {
		metrics := s.storage.Export()
		if err := s.fileStorage.Save(metrics); err != nil {
			s.logger.Fatalf("Sync save failed")
		}
	}
}
