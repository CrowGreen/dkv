package auth

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type KeyStore struct {
	mu   sync.RWMutex
	keys map[string]string // apiKey -> clientName
}

func NewKeyStore() *KeyStore {
	return &KeyStore{keys: make(map[string]string)}
}

func (ks *KeyStore) AddKey(apiKey, clientName string) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.keys[apiKey] = clientName
}

func (ks *KeyStore) Validate(apiKey string) (string, bool) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	name, ok := ks.keys[apiKey]
	return name, ok
}

func UnaryInterceptor(ks *KeyStore) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		clientName, err := authenticate(ctx, ks)
		if err != nil {
			return nil, err
		}
		// attach client name to context
		ctx = context.WithValue(ctx, "client", clientName)
		return handler(ctx, req)
	}
}

func StreamInterceptor(ks *KeyStore) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		_, err := authenticate(ss.Context(), ks)
		if err != nil {
			return err
		}
		return handler(srv, ss)
	}
}

func authenticate(ctx context.Context, ks *KeyStore) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	keys := md.Get("x-api-key")
	if len(keys) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing api key")
	}

	clientName, valid := ks.Validate(keys[0])
	if !valid {
		return "", status.Error(codes.PermissionDenied, fmt.Sprintf("invalid api key"))
	}
	return clientName, nil
}
