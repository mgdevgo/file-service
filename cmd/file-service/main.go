package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	pgxtx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	tx "github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"file-service/internal/api"
	"file-service/internal/ratelimit"
	"file-service/internal/server"
	"file-service/internal/service"
	"file-service/internal/storage/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// if err := godotenv.Load(); err != nil {
	// 	logger.Error("failed to load .env file", "error", err)
	// }

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := postgres.New(ctx, os.Getenv("DATABASE_URL"), os.Getenv("MIGRATIONS_PATH"))
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	transaction := tx.Must(pgxtx.NewDefaultFactory(db))

	metaStorage := postgres.NewFileMetaStorage(db, pgxtx.DefaultCtxGetter, logger)
	fileService, err := service.NewDiskFileService(os.Getenv("FILES_UPLOAD_PATH"), metaStorage, transaction, logger)
	if err != nil {
		logger.Error("failed to create file service", "error", err)
		os.Exit(1)
	}

	fileServer := server.NewFileServer(fileService)
	limiter := ratelimit.NewRequestLimiter()
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			recovery.UnaryServerInterceptor(
				recovery.WithRecoveryHandler(
					func(p any) (err error) {
						logger.Error("Recovered from panic", slog.Any("panic", p))
						return status.Errorf(codes.Internal, "internal error")
					}),
			),
			logging.UnaryServerInterceptor(
				logging.LoggerFunc(
					func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
						logger.Log(ctx, slog.Level(lvl), msg, fields...)
					},
				),
				logging.WithLogOnEvents(logging.PayloadReceived, logging.PayloadSent),
			),
			limiter.UnaryInterceptor,
		),
	)
	reflection.Register(server)

	api.RegisterFileServiceServer(server, fileServer)

	listen, err := net.Listen("tcp", ":8080")
	if err != nil {
		logger.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	interrupt := make(chan os.Signal, 1)
	shutdownSignals := []os.Signal{
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
	}
	signal.Notify(interrupt, shutdownSignals...)

	go func() {
		logger.Info("gRPC server started on " + ":8080")
		if err := server.Serve(listen); err != nil {
			logger.Error("failed to serve", "error", err)
			os.Exit(1)
		}
	}()

	select {
	case killSignal := <-interrupt:
		logger.Info("Recived interrupt signal", "signal", killSignal)
		cancel()
	case <-ctx.Done():
		logger.Info("Context cancelled", "error", ctx.Err())
	}

	server.GracefulStop()
	logger.Info("gRPC server stopped")
}
