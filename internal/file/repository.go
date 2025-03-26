package file

import (
	"context"

	"github.com/google/uuid"
)

type FileMetaRepository interface {
	Save(ctx context.Context, meta *FileMeta) error
	FindAll(ctx context.Context, page Page) ([]*FileMeta, error)
	FindById(ctx context.Context, id uuid.UUID) (*FileMeta, error)
}
