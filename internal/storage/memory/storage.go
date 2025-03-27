package memory

import (
	"context"
	"file-service/internal/file"
	"sync"

	"github.com/google/uuid"
)

// Storage implements the file.FileMetaRepository interface using in-memory storage
type Storage struct {
	mu      sync.RWMutex
	entries map[string]file.FileMeta
}

// New creates a new in-memory storage instance
func New() *Storage {
	return &Storage{
		entries: make(map[string]file.FileMeta),
	}
}

// Save adds a new file meta
func (s *Storage) Save(ctx context.Context, entry file.FileMeta) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries[entry.ID.String()] = entry
	return nil
}

// FindById retrieves a file meta by ID
func (s *Storage) FindById(ctx context.Context, id uuid.UUID) (*file.FileMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.entries[id.String()]
	if !exists {
		return nil, file.ErrFileNotFound
	}

	return &entry, nil
}

// FindAll retrieves all file meta
func (s *Storage) FindAll(ctx context.Context, page file.Page) ([]file.FileMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]file.FileMeta, 0, len(s.entries))
	for _, entry := range s.entries {
		entries = append(entries, entry)
	}

	return entries, nil
}

// Close performs any necessary cleanup
func (s *Storage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = make(map[string]file.FileMeta)
	return nil
}
