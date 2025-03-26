package server

import (
	"context"

	"tages-go/internal/api"
	"tages-go/internal/file"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type FileServer struct {
	api.UnimplementedFileServiceServer

	fileService file.FileService
}

func NewFileServer(fileService file.FileService) *FileServer {
	return &FileServer{fileService: fileService}
}

func (s *FileServer) UploadFile(ctx context.Context, request *api.UploadFileRequest) (*api.UploadFileResponse, error) {
	err := s.fileService.UploadFile(ctx, request.Filename, request.Data)
	if err != nil {
		return nil, err
	}

	return &api.UploadFileResponse{}, nil
}

func (s *FileServer) DownloadFile(ctx context.Context, request *api.DownloadFileRequest) (*api.DownloadFileResponse, error) {
	filename, data, err := s.fileService.DownloadFile(ctx, request.FileId)
	if err != nil {
		return nil, err
	}

	return &api.DownloadFileResponse{
		Filename: filename,
		Data:     data,
	}, nil
}

func (s *FileServer) ViewFiles(ctx context.Context, request *api.ViewFilesRequest) (*api.ViewFilesResponse, error) {
	files, err := s.fileService.ViewFilesMetadata(ctx)
	if err != nil {
		return nil, err
	}

	responseFiles := make([]*api.ViewFilesResponse_FileInfo, len(files))
	for i, file := range files {
		responseFiles[i] = &api.ViewFilesResponse_FileInfo{
			Filename:  file.Filename,
			CreatedAt: timestamppb.New(file.CreatedAt),
			UpdatedAt: timestamppb.New(file.UpdatedAt),
		}
	}

	return &api.ViewFilesResponse{
		Files: responseFiles,
	}, nil
}
