package file

import "context"

type FileService interface {
	UploadFile(ctx context.Context, fileName string, fileData []byte) (string, error)
	DownloadFile(ctx context.Context, fileId string) (string, []byte, error)
	ViewFilesMetadata(ctx context.Context, page Page) ([]*FileMeta, error)
}
