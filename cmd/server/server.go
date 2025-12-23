package main

type Server struct {
	storage     Storage
	fileStorage *FileStorage
	syncSave    bool
}

func (s *Server) saveIfNeeded() {
	if s.syncSave {
		metrics := s.storage.Export()
		_ = s.fileStorage.Save(metrics)
	}
}
