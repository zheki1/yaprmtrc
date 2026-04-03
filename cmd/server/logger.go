package main

import "go.uber.org/zap"

// Logger — интерфейс логирования, используемый сервером и мидлварами.
type Logger interface {
	Infow(msg string, fields ...any)
	Fatalf(template string, args ...interface{})
	Error(args ...interface{})
	Errorf(template string, args ...interface{})
}

// NewLogger создаёт production-логгер на базе zap.
func NewLogger() (*zap.SugaredLogger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	return logger.Sugar(), nil
}
