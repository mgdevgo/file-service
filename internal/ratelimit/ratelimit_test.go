package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestRequestLimiter_UnaryInterceptor(t *testing.T) {
	createContext := func(clientID string) context.Context {
		md := metadata.New(map[string]string{"client-id": clientID})
		return metadata.NewIncomingContext(context.Background(), md)
	}

	createHandler := func(ctx context.Context, wait bool) grpc.UnaryHandler {
		return func(ctx context.Context, req any) (any, error) {
			if wait {
				<-ctx.Done()
			}
			return "success", nil
		}
	}

	t.Run("should allow requests within limits", func(t *testing.T) {
		limiter := NewRequestLimiter()

		ctx := createContext("test-client")
		info := &grpc.UnaryServerInfo{
			FullMethod: "/file.FileService/Upload",
		}

		errorGroup := errgroup.Group{}
		var response any
		for range 10 {
			errorGroup.Go(func() error {
				resp, err := limiter.UnaryInterceptor(ctx, nil, info, createHandler(ctx, false))
				if err != nil {
					return err
				}
				response = resp
				return nil
			})
		}

		err := errorGroup.Wait()
		require.NoError(t, err)
		assert.Equal(t, "success", response)
		statusErr, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.OK, statusErr.Code())
	})

	t.Run("should reject requests exceeding limits", func(t *testing.T) {
		limiter := NewRequestLimiter()

		ctx := createContext("test-client")
		info := &grpc.UnaryServerInfo{
			FullMethod: "/file.FileService/Upload",
		}

		ctx, cancel := context.WithCancel(ctx)

		for i := range 10 {
			go func() {
				t.Logf("Request %d", i)
				_, err := limiter.UnaryInterceptor(ctx, nil, info, createHandler(ctx, true))
				require.NoError(t, err)
			}()
		}
		time.Sleep(1 * time.Second)

		// This request should be rejected
		_, err := limiter.UnaryInterceptor(ctx, nil, info, createHandler(ctx, false))
		t.Logf("error: %v", err)
		require.Error(t, err)
		statusErr, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.ResourceExhausted, statusErr.Code())
		require.Equal(t, "too many concurrent requests", statusErr.Message())

		cancel()
	})

	t.Run("should reject requests without client-id", func(t *testing.T) {
		limiter := NewRequestLimiter()

		ctx := createContext("")
		info := &grpc.UnaryServerInfo{
			FullMethod: "/file.FileService/Upload",
		}

		resp, err := limiter.UnaryInterceptor(ctx, nil, info, createHandler(ctx, true))
		assert.Error(t, err)
		assert.Nil(t, resp)
		statusErr, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, statusErr.Code())
		assert.Equal(t, "client-id required", statusErr.Message())
	})
}
