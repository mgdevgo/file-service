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
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"file-service/internal/api"
	"file-service/internal/ratelimit"
	"file-service/internal/server"
	"file-service/internal/service"
	"file-service/internal/storage/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := postgres.New(context.Background(), "postgres://postgres:postgres@localhost:5432/postgres")
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	transaction := tx.Must(pgxtx.NewDefaultFactory(db))

	metaStorage := postgres.NewFileMetaStorage(db, pgxtx.DefaultCtxGetter, logger)
	fileService := service.NewDiskFileService("./storage", metaStorage, transaction, logger)

	fileServer := server.NewFileServer(fileService)
	limiter := ratelimit.NewRequestLimiter()
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			// logging.UnaryServerInterceptor(InterceptorLogger(logger), opts...),)
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
