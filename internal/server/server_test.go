package server_test

import (
	"context"
	"log"
	"log/slog"
	"net"
	"os"
	"testing"

	pgxtx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	tx "github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"file-service/internal/api"
	"file-service/internal/server"
	"file-service/internal/service"
	"file-service/internal/storage/postgres"
)

func TestFileServer_UploadFile(t *testing.T) {
	client := setupTest(t)

	testImage, err := os.ReadFile("../../testdata/test_image.jpg")
	require.NoError(t, err)

	tests := []struct {
		name    string
		request *api.UploadFileRequest
		want    *api.UploadFileResponse
		errCode codes.Code
		errMsg  string
	}{
		{
			name: "Success",
			request: &api.UploadFileRequest{
				Filename: "test_image.jpg",
				Data:     testImage,
			},
			want:    &api.UploadFileResponse{},
			errCode: codes.OK,
		},
		{
			name: "Empty file",
			request: &api.UploadFileRequest{
				Filename: "test_image.jpg",
				Data:     []byte{},
			},
			errCode: codes.InvalidArgument,
			errMsg:  "File content can't be empty",
		},
		{
			name: "File without name",
			request: &api.UploadFileRequest{
				Filename: "",
				Data:     testImage,
			},
			errCode: codes.InvalidArgument,
			errMsg:  "File name can't be empty",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := client.UploadFile(context.Background(), test.request)
			if test.errCode != codes.OK {
				require.Equal(t, test.errCode, status.Code(err))
				require.Contains(t, err.Error(), test.errMsg)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, response.FileId)
			_, err = uuid.Parse(response.FileId)
			require.NoError(t, err)

		})
	}
}

func TestFileServer_DownloadFile(t *testing.T) {
	client := setupTest(t)

	testImage, err := os.ReadFile("../../testdata/test_image.jpg")
	require.NoError(t, err)

	uploadResponse, err := client.UploadFile(context.Background(), &api.UploadFileRequest{
		Filename: "test_image.jpg",
		Data:     testImage,
	})
	require.NoError(t, err)

	response, err := client.DownloadFile(context.Background(), &api.DownloadFileRequest{
		FileId: uploadResponse.FileId,
	})
	require.NoError(t, err)
	require.NotEmpty(t, response.Data)
	require.Equal(t, testImage, response.Data)
}

func TestFileServer_ViewFiles(t *testing.T) {
	client := setupTest(t)

	testImage, err := os.ReadFile("../../testdata/test_image.jpg")
	require.NoError(t, err)

	_, err = client.UploadFile(context.Background(), &api.UploadFileRequest{
		Filename: "test_image.jpg",
		Data:     testImage,
	})
	require.NoError(t, err)

	response, err := client.ViewFiles(context.Background(), &api.ViewFilesRequest{
		Offset: 0,
		Limit:  10,
	})
	require.NoError(t, err)
	require.NotEmpty(t, response.Files)
}

func setupTest(t *testing.T) api.FileServiceClient {
	t.Helper()

	ctx := context.Background()
	logger := slog.Default()

	db, err := postgres.New(ctx, "postgres://postgres:postgres@localhost:5432/postgres", "../../migrations")
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Close()
	})

	storagePath := t.TempDir()
	metaStorage := postgres.NewFileMetaStorage(db, pgxtx.DefaultCtxGetter, logger)

	txManager := tx.Must(pgxtx.NewDefaultFactory(db))

	fileService, err := service.NewDiskFileService(storagePath, metaStorage, txManager, logger)
	require.NoError(t, err)

	fileServer := server.NewFileServer(fileService)

	buffer := 101024 * 1024
	listener := bufconn.Listen(buffer)
	t.Cleanup(func() {
		listener.Close()
	})

	server := grpc.NewServer()
	t.Cleanup(func() {
		server.Stop()
	})

	api.RegisterFileServiceServer(server, fileServer)
	go func() {
		if err := server.Serve(listener); err != nil {
			log.Fatalf("error serving server: %v", err)
		}
	}()

	conn, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(
			func(ctx context.Context, s string) (net.Conn, error) {
				return listener.Dial()
			}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err, "error creating client")
	t.Cleanup(func() {
		conn.Close()
	})

	client := api.NewFileServiceClient(conn)

	return client
}
