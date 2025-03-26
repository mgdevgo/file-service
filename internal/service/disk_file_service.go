package service

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	tx "github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/google/uuid"

	"file-service/internal/api"
	"file-service/internal/file"
)

const (
	MAX_FILE_SIZE = 1000 * 1024 * 1024 // 1GB

	READ_TIMEOUT     = 15 * time.Second
	SHUTDOWN_TIMEOUT = 15 * time.Second
)

type DiskFileService struct {
	api.UnimplementedFileServiceServer

	storagePath string
	meta        file.FileMetaRepository
	transaction *tx.Manager
	logger      *slog.Logger
}

func NewDiskFileService(storagePath string, metaRepo file.FileMetaRepository, transaction *tx.Manager, logger *slog.Logger) *DiskFileService {
	return &DiskFileService{
		storagePath: storagePath,
		meta:        metaRepo,
		transaction: transaction,
		logger:      logger,
	}
}

func (service *DiskFileService) UploadFile(ctx context.Context, fileName string, fileData []byte) (string, error) {
	meta := file.NewFileMeta(uuid.New(), fileName, fileData)
	file := file.NewFile(fileData, meta)

	err := service.transaction.Do(ctx,
		func(ctx context.Context) error {
			if err := service.meta.Save(ctx, &file.Meta); err != nil {
				return err
			}

			if err := service.writeToDisk(ctx, file); err != nil {
				return err
			}

			return nil
		},
	)

	if err != nil {
		return "", err
	}

	return file.Meta.ID.String(), nil
}

func (service *DiskFileService) writeToDisk(ctx context.Context, file *file.File) error {
	path := filepath.Join(service.storagePath, file.Meta.Hash[:2], file.Meta.Hash[2:4], file.Meta.Hash)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(path, file.Content, 0644); err != nil {
		return err
	}

	return nil
}

func (service *DiskFileService) DownloadFile(ctx context.Context, fileId string) (string, []byte, error) {
	fileUUID, err := uuid.Parse(fileId)
	if err != nil {
		return "", nil, err
	}

	meta, err := service.meta.FindById(ctx, fileUUID)
	if err != nil {
		return "", nil, err
	}

	filePath := filepath.Join(service.storagePath, meta.Filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", nil, err
	}

	return meta.Filename, data, nil
}

func (service *DiskFileService) ViewFilesMetadata(ctx context.Context) ([]*file.FileMeta, error) {
	files, err := service.meta.FindAll(ctx, file.Page{})
	if err != nil {
		return nil, err
	}

	return files, nil
}
