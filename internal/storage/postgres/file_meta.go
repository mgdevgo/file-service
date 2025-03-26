package postgres

import (
	"context"
	"log/slog"

	pgxtx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"tages-go/internal/file"
)

type FileMetaStorage struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	tx     *pgxtx.CtxGetter
}

func NewFileMetaStorage(pool *pgxpool.Pool, transaction *pgxtx.CtxGetter, logger *slog.Logger) *FileMetaStorage {
	return &FileMetaStorage{
		pool:   pool,
		tx:     transaction,
		logger: logger,
	}
}

func (s *FileMetaStorage) Save(ctx context.Context, file *file.FileMeta) error {
	query := `
	INSERT INTO file_meta (id, filename, hash, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (id) DO UPDATE
	SET filename = EXCLUDED.filename,
		hash = EXCLUDED.hash,
		created_at = EXCLUDED.created_at,
		updated_at = EXCLUDED.updated_at
	RETURNING id`

	db := s.tx.DefaultTrOrDB(ctx, s.pool)

	return db.QueryRow(ctx, query, file.ID, file.Filename, file.Hash, file.CreatedAt, file.UpdatedAt).Scan(&file.ID)
}

func (s *FileMetaStorage) FindAll(ctx context.Context, page file.Page) ([]*file.FileMeta, error) {
	query := `
	SELECT id, filename, hash, created_at, updated_at
	FROM file_meta
	ORDER BY updated_at DESC, created_at DESC
	LIMIT $1 OFFSET $2`

	db := s.tx.DefaultTrOrDB(ctx, s.pool)

	rows, err := db.Query(ctx, query, page.Size, (page.Number-1)*page.Size)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := make([]*file.FileMeta, 0)
	for rows.Next() {
		var meta file.FileMeta
		err = rows.Scan(&meta.ID, &meta.Filename, &meta.Hash, &meta.CreatedAt, &meta.UpdatedAt)
		if err != nil {
			return nil, err
		}
		files = append(files, &meta)
	}

	return files, nil
}

func (s *FileMetaStorage) FindById(ctx context.Context, id uuid.UUID) (*file.FileMeta, error) {
	query := `
	SELECT id, filename, hash, created_at, updated_at
	FROM file_meta
	WHERE id = $1`

	db := s.tx.DefaultTrOrDB(ctx, s.pool)

	var meta file.FileMeta
	err := db.QueryRow(ctx, query, id).Scan(&meta.ID, &meta.Filename, &meta.Hash, &meta.CreatedAt, &meta.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}
