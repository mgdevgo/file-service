package ratelimit

import (
	"context"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	MAX_CONCURRENT_UPLOADS   = 10
	MAX_CONCURRENT_DOWNLOADS = 100
)

type RequestLimiter struct {
	clients map[string]map[string]int // client-id -> method -> active requests
	limits  map[string]int
	mutex   sync.Mutex
}

func NewRequestLimiter() *RequestLimiter {
	return &RequestLimiter{
		clients: make(map[string]map[string]int),
		limits: map[string]int{
			"/file.FileService/Upload":   MAX_CONCURRENT_UPLOADS,
			"/file.FileService/Download": MAX_CONCURRENT_DOWNLOADS,
		},
	}
}

func (limiter *RequestLimiter) UnaryInterceptor(
	ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
) (any, error) {
	method := info.FullMethod
	limit, exists := limiter.limits[method]
	if !exists {
		return handler(ctx, req)
	}

	// IP strategy
	//
	// p, ok := peer.FromContext(ctx)
	// if !ok {
	// 	return nil, status.Error(codes.Internal, "failed to get client IP")
	// }
	// clientIP := p.Addr.String()

	// Client ID strategy
	meta, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "failed to get metadata from context")
	}

	values := meta.Get("client-id")
	if len(values) <= 0 {
		return nil, status.Error(codes.Unauthenticated, "client-id required")
	}
	clientId := values[0]

	limiter.mutex.Lock()
	activeRequests, ok := limiter.clients[clientId]
	if !ok {
		activeRequests = make(map[string]int)
		limiter.clients[clientId] = activeRequests
	}
	if activeRequests[method] >= limit {
		limiter.mutex.Unlock()
		return nil, status.Error(codes.ResourceExhausted, "too many concurrent requests")
	}
	limiter.clients[clientId][method]++
	limiter.mutex.Unlock()

	resp, err := handler(ctx, req)

	limiter.mutex.Lock()
	limiter.clients[clientId][method]--
	limiter.mutex.Unlock()

	return resp, err
}
