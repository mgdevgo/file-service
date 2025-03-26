package file

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

type File struct {
	Content []byte
	Meta    FileMeta
}

func NewFile(data []byte, meta FileMeta) *File {
	return &File{
		Content: data,
		Meta:    meta,
	}
}

type FileMeta struct {
	ID        uuid.UUID
	Filename  string
	Hash      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewFileMeta(fileId uuid.UUID, filename string, data []byte) FileMeta {
	return FileMeta{
		ID:        fileId,
		Filename:  filename,
		Hash:      hashFile(data),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (file *File) Update() {
	file.Meta.UpdatedAt = time.Now()
}

func hashFile(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
