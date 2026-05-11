package main

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	// TrustedSubnetHeaderKey — ключ заголовка метаданных для IP-адреса агента.
	TrustedSubnetHeaderKey = "x-real-ip"
)

// NewTrustedSubnetInterceptor создаёт UnaryServerInterceptor для проверки подсети.
// Перехватчик проверяет, принадлежит ли IP-адрес агента (из метаданных x-real-ip)
// доверенной подсети. Если подсеть не настроена, проверка пропускается.
func NewTrustedSubnetInterceptor(trustedSubnet string, logger Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Если TrustedSubnet не установлен, пропускаем проверку
		if trustedSubnet == "" {
			return handler(ctx, req)
		}

		// Получаем метаданные из контекста
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			logger.Error("no metadata in incoming context")
			return nil, status.Error(codes.PermissionDenied, "x-real-ip header is required")
		}

		// Получаем IP-адрес из метаданных
		realIPValues := md.Get(TrustedSubnetHeaderKey)
		if len(realIPValues) == 0 {
			logger.Error("x-real-ip header not found in metadata")
			return nil, status.Error(codes.PermissionDenied, "x-real-ip header is required")
		}

		realIP := realIPValues[0]

		// Парсим доверенную подсеть
		_, trustedNetwork, err := net.ParseCIDR(trustedSubnet)
		if err != nil {
			logger.Error("failed to parse trusted subnet", err.Error())
			return nil, status.Error(codes.Internal, "server configuration error")
		}

		// Парсим IP-адрес
		agentIP := net.ParseIP(realIP)
		if agentIP == nil {
			logger.Error("invalid x-real-ip format", realIP)
			return nil, status.Error(codes.PermissionDenied, fmt.Sprintf("invalid x-real-ip format: %s", realIP))
		}

		// Проверяем, входит ли IP в доверенную подсеть
		if !trustedNetwork.Contains(agentIP) {
			logger.Error("agent IP is not in trusted subnet", fmt.Sprintf("%s not in %s", realIP, trustedSubnet))
			return nil, status.Error(codes.PermissionDenied, "agent IP is not in trusted subnet")
		}

		// IP проверен, передаём запрос дальше
		return handler(ctx, req)
	}
}
