package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/example/forumgo/internal/logger"
	"github.com/example/forumgo/internal/model"
)

// MemoryBoardStore is an in-memory board store with JSON persistence.
type MemoryBoardStore struct {
	mu     sync.RWMutex
	items  map[string]*model.Board
	slugIdx map[string]string // slug -> id
	path   string
	dirty  bool
	log    *logger.Logger
	stop   chan struct{}
	done   chan struct{}
}

// NewMemoryBoardStore creates a MemoryBoardStore.
func NewMemoryBoardStore(path string, log *logger.Logger, interval time.Duration) (*MemoryBoardStore, error) {
	if log == nil {
		log = logger.Default()
	}
	s := &MemoryBoardStore{
		items:   make(map[string]*model.Board),
		slugIdx: make(map[string]string),
		path:    path,
		log:     log,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	if path != "" {
		if err := s.load(); err != nil {
			log.Warn("board store load failed", "err", err.Error())
		}
		go s.saveLoop(interval)
	} else {
		close(s.done)
	}
	return s, nil
}

func (s *MemoryBoardStore) Create(ctx context.Context, b *model.Board) error {
	if b == nil || b.ID == "" {
		return model.ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.items[b.ID]; exists {
		return model.ErrAlreadyExists
	}
	if b.Slug != "" {
		if _, used := s.slugIdx[b.Slug]; used {
			return fmt.Errorf("%w: slug already used", model.ErrConflict)
		}
		s.slugIdx[b.Slug] = b.ID
	}
	s.items[b.ID] = b.Clone()
	s.dirty = true
	return nil
}

func (s *MemoryBoardStore) GetByID(ctx context.Context, id string) (*model.Board, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.items[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return b.Clone(), nil
}

func (s *MemoryBoardStore) GetBySlug(ctx context.Context, slug string) (*model.Board, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.slugIdx[slug]
	if !ok {
		return nil, model.ErrNotFound
	}
	b, ok := s.items[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return b.Clone(), nil
}

func (s *MemoryBoardStore) Update(ctx context.Context, b *model.Board) error {
	if b == nil || b.ID == "" {
		return model.ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	orig, ok := s.items[b.ID]
	if !ok {
		return model.ErrNotFound
	}
	// If the slug changed, reconcile the slug index. Reject conflicts with
	// other boards: a slug that only differs by spaces/case before
	// normalization (already applied upstream) must not silently overwrite the
	// existing board's index entry.
	if b.Slug != orig.Slug {
		if b.Slug != "" {
			if ownerID, used := s.slugIdx[b.Slug]; used && ownerID != b.ID {
				return fmt.Errorf("%w: slug already used", model.ErrConflict)
			}
		}
		delete(s.slugIdx, orig.Slug)
		if b.Slug != "" {
			s.slugIdx[b.Slug] = b.ID
		}
	}
	s.items[b.ID] = b.Clone()
	s.dirty = true
	return nil
}

func (s *MemoryBoardStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.items[id]
	if !ok {
		return model.ErrNotFound
	}
	delete(s.slugIdx, b.Slug)
	delete(s.items, id)
	s.dirty = true
	return nil
}

func (s *MemoryBoardStore) List(ctx context.Context, f model.BoardFilter) ([]*model.Board, error) {
	s.mu.RLock()
	var out []*model.Board
	for _, b := range s.items {
		if f.Matches(b) {
			out = append(out, b.Clone())
		}
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].DisplayOrder != out[j].DisplayOrder {
			return out[i].DisplayOrder < out[j].DisplayOrder
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (s *MemoryBoardStore) Close() error {
	if s.path == "" {
		return nil
	}
	s.mu.Lock()
	close(s.stop)
	s.mu.Unlock()
	<-s.done
	return s.flush()
}

func (s *MemoryBoardStore) saveLoop(interval time.Duration) {
	defer close(s.done)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.mu.RLock()
			dirty := s.dirty
			s.mu.RUnlock()
			if dirty {
				s.flush()
			}
		}
	}
}

func (s *MemoryBoardStore) flush() error {
	if s.path == "" {
		return nil
	}
	s.mu.Lock()
	snapshot := make([]*model.Board, 0, len(s.items))
	for _, b := range s.items {
		snapshot = append(snapshot, b.Clone())
	}
	s.dirty = false
	s.mu.Unlock()
	sort.Slice(snapshot, func(i, j int) bool {
		if snapshot[i].DisplayOrder != snapshot[j].DisplayOrder {
			return snapshot[i].DisplayOrder < snapshot[j].DisplayOrder
		}
		return snapshot[i].ID < snapshot[j].ID
	})
	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal boards: %w", err)
	}
	dir := filepath.Dir(s.path)
	if dir != "" && dir != "." {
		os.MkdirAll(dir, 0o755) // #nosec G301
	}
	tmp, err := os.CreateTemp(dir, ".forumgo-boards-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(payload); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, s.path)
}

func (s *MemoryBoardStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	var items []*model.Board
	if err := json.Unmarshal(data, &items); err != nil {
		return fmt.Errorf("corrupt board data: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, b := range items {
		if b != nil && b.ID != "" {
			s.items[b.ID] = b
			if b.Slug != "" {
				s.slugIdx[b.Slug] = b.ID
			}
		}
	}
	return nil
}
