package main

type Server struct {
	storage     Storage
	logger      Logger
	fileStorage *FileStorage
	syncSave    bool
}

func (s *Server) saveIfNeeded() {
	if s.syncSave {
		metrics := s.storage.Export()
		if err := s.fileStorage.Save(metrics); err != nil {
			s.logger.Fatalf("Sync save failed")
		}
	}
}
