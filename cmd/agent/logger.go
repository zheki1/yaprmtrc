package main

import "go.uber.org/zap"

type Logger interface {
	Infow(msg string, fields ...any)
	Fatalf(template string, args ...interface{})
	Error(args ...interface{})
}

func NewLogger() (*zap.SugaredLogger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	return logger.Sugar(), nil
}
