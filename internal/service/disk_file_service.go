package service

import (
	"context"
	"errors"
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

	DEFAULT_FILES_UPLOAD_PATH = "./uploads"
)

type DiskFileService struct {
	api.UnimplementedFileServiceServer

	uploadPath  string
	meta        file.FileMetaRepository
	transaction *tx.Manager
	logger      *slog.Logger
}

func NewDiskFileService(uploadPath string, metaRepo file.FileMetaRepository, transaction *tx.Manager, logger *slog.Logger) (*DiskFileService, error) {
	if uploadPath == "" {
		uploadPath = DEFAULT_FILES_UPLOAD_PATH
	}

	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		return nil, err
	}

	return &DiskFileService{
		uploadPath:  uploadPath,
		meta:        metaRepo,
		transaction: transaction,
		logger:      logger,
	}, nil
}

func (service *DiskFileService) UploadFile(ctx context.Context, fileName string, fileData []byte) (string, error) {
	meta, err := file.NewFileMeta(uuid.New(), fileName, fileData)
	if err != nil {
		return "", err
	}
	file, err := file.NewFile(fileData, meta)
	if err != nil {
		return "", err
	}

	if err := service.transaction.Do(ctx,
		func(ctx context.Context) error {
			if err := service.meta.Save(ctx, &file.Meta); err != nil {
				return err
			}

			if err := service.writeToDisk(ctx, file); err != nil {
				return err
			}

			return nil
		},
	); err != nil {
		return "", err
	}

	return file.Meta.ID.String(), nil
}

func (service *DiskFileService) writeToDisk(ctx context.Context, file *file.File) error {
	path := createFilePath(service.uploadPath, file.Meta.Hash)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(path, file.Content, 0644); err != nil {
		return err
	}

	return nil
}

func createFilePath(base, hash string) string {
	return filepath.Join(base, hash[:2], hash[2:4], hash)
}

func (service *DiskFileService) DownloadFile(ctx context.Context, fileId string) (string, []byte, error) {
	fileUUID, err := uuid.Parse(fileId)
	if err != nil {
		return "", nil, file.ErrFileIdEmpty
	}

	meta, err := service.meta.FindById(ctx, fileUUID)
	if err != nil {
		return "", nil, err
	}

	filePath := createFilePath(service.uploadPath, meta.Hash)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil, file.ErrFileNotFound
		}
		return "", nil, err
	}

	return meta.Filename, data, nil
}

func (service *DiskFileService) ViewFilesMetadata(ctx context.Context, page file.Page) ([]*file.FileMeta, error) {
	files, err := service.meta.FindAll(ctx, page)
	if err != nil {
		return nil, err
	}

	return files, nil
}
