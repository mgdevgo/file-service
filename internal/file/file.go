package file

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrFile              = errors.New("file")
	ErrFileNotFound      = fmt.Errorf("%w: not found", ErrFile)
	ErrFileAlreadyExists = fmt.Errorf("%w: already exists", ErrFile)
	ErrFileEmpty         = fmt.Errorf("%w: content is empty", ErrFile)
	ErrFileNameEmpty     = fmt.Errorf("%w: name is empty", ErrFile)
	ErrFileIdEmpty       = fmt.Errorf("%w: id is empty", ErrFile)
)

type File struct {
	Content []byte
	Meta    FileMeta
}

func NewFile(data []byte, meta FileMeta) (*File, error) {
	if len(data) == 0 {
		return nil, ErrFileEmpty
	}
	return &File{
		Content: data,
		Meta:    meta,
	}, nil
}

type FileMeta struct {
	ID        uuid.UUID
	Filename  string
	Hash      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewFileMeta(fileId uuid.UUID, filename string, data []byte) (FileMeta, error) {
	if filename == "" {
		return FileMeta{}, ErrFileNameEmpty
	}
	if len(data) == 0 {
		return FileMeta{}, ErrFileEmpty
	}
	return FileMeta{
		ID:        fileId,
		Filename:  filename,
		Hash:      hashFile(data),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (file *File) Update() {
	file.Meta.UpdatedAt = time.Now()
}

func hashFile(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
