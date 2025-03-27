package server

import (
	"context"
	"errors"
	"fmt"

	"file-service/internal/api"
	"file-service/internal/file"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	var response *api.UploadFileResponse
	id, err := s.fileService.UploadFile(ctx, request.Filename, request.Data)
	if err != nil {
		if errors.Is(err, file.ErrFileEmpty) {
			return nil, status.Errorf(codes.InvalidArgument, "File content can't be empty")
		}
		if errors.Is(err, file.ErrFileNameEmpty) {
			return nil, status.Errorf(codes.InvalidArgument, "File name can't be empty")
		}
		status, err := status.New(codes.Internal, "Failed to upload file").
			WithDetails(&errdetails.ErrorInfo{Reason: err.Error()})
		if err != nil {
			return nil, fmt.Errorf("unexpected error attaching error detail: %w", err)
		}
		return nil, status.Err()
	}

	response = &api.UploadFileResponse{
		FileId: id,
	}

	return response, nil
}

func (s *FileServer) DownloadFile(ctx context.Context, request *api.DownloadFileRequest) (*api.DownloadFileResponse, error) {
	filename, data, err := s.fileService.DownloadFile(ctx, request.FileId)
	if err != nil {
		if errors.Is(err, file.ErrFileNotFound) {
			return nil, status.Errorf(codes.NotFound, "File with id %s not found", request.FileId)
		}

		if errors.Is(err, file.ErrFileIdEmpty) {
			return nil, status.Errorf(codes.InvalidArgument, "File id can't be empty")
		}

		status, err := status.New(
			codes.Internal,
			fmt.Sprintf("Failed to download file with id %s", request.FileId),
		).WithDetails(&errdetails.ErrorInfo{Reason: err.Error()})
		if err != nil {
			return nil, fmt.Errorf("unexpected error attaching error detail: %w", err)
		}
		return nil, status.Err()
	}

	return &api.DownloadFileResponse{
		Filename: filename,
		Data:     data,
	}, nil
}

func (s *FileServer) ViewFiles(ctx context.Context, request *api.ViewFilesRequest) (*api.ViewFilesResponse, error) {
	files, err := s.fileService.ViewFilesMetadata(ctx, file.NewPage(int(request.Offset), int(request.Limit)))
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
