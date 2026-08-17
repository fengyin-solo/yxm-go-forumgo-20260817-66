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

// MemoryCommentStore is an in-memory comment store with soft delete and JSON persistence.
type MemoryCommentStore struct {
	mu       sync.RWMutex
	items    map[string]*model.Comment
	threadIdx map[string][]string // threadID -> comment IDs
	path     string
	dirty    bool
	log      *logger.Logger
	stop     chan struct{}
	done     chan struct{}
}

// NewMemoryCommentStore creates a MemoryCommentStore.
func NewMemoryCommentStore(path string, log *logger.Logger, interval time.Duration) (*MemoryCommentStore, error) {
	if log == nil {
		log = logger.Default()
	}
	s := &MemoryCommentStore{
		items:    make(map[string]*model.Comment),
		threadIdx: make(map[string][]string),
		path:     path,
		log:      log,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
	if path != "" {
		if err := s.load(); err != nil {
			log.Warn("comment store load failed", "err", err.Error())
		}
		go s.saveLoop(interval)
	} else {
		close(s.done)
	}
	return s, nil
}

func (s *MemoryCommentStore) Create(ctx context.Context, c *model.Comment) error {
	if c == nil || c.ID == "" {
		return model.ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.items[c.ID]; exists {
		return model.ErrAlreadyExists
	}
	s.items[c.ID] = c.Clone()
	s.threadIdx[c.ThreadID] = append(s.threadIdx[c.ThreadID], c.ID)
	s.dirty = true
	return nil
}

func (s *MemoryCommentStore) GetByID(ctx context.Context, id string) (*model.Comment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.items[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return c.Clone(), nil
}

func (s *MemoryCommentStore) Update(ctx context.Context, c *model.Comment) error {
	if c == nil || c.ID == "" {
		return model.ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[c.ID]; !ok {
		return model.ErrNotFound
	}
	s.items[c.ID] = c.Clone()
	s.dirty = true
	return nil
}

func (s *MemoryCommentStore) SoftDelete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.items[id]
	if !ok {
		return model.ErrNotFound
	}
	c.IsDeleted = true
	s.dirty = true
	return nil
}

func (s *MemoryCommentStore) List(ctx context.Context, f model.CommentFilter) ([]*model.Comment, error) {
	s.mu.RLock()
	var ids []string
	if f.ThreadID != "" {
		ids = s.threadIdx[f.ThreadID]
	} else {
		ids = make([]string, 0, len(s.items))
		for id := range s.items {
			ids = append(ids, id)
		}
	}
	var out []*model.Comment
	for _, id := range ids {
		c := s.items[id]
		if !c.Visible() {
			continue
		}
		if f.ParentID != nil {
			hasParent := c.ParentID != nil
			if *f.ParentID && !hasParent {
				continue
			}
			if !*f.ParentID && hasParent {
				continue
			}
		}
		if f.AuthorID != "" && c.AuthorID != f.AuthorID {
			continue
		}
		out = append(out, c.Clone())
	}
	s.mu.RUnlock()
	switch f.SortBy {
	case "oldest":
		sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	case "score":
		sort.Slice(out, func(i, j int) bool { return out[i].Score() > out[j].Score() })
	default: // "newest"
		sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	}
	return out, nil
}

func (s *MemoryCommentStore) CountByThread(ctx context.Context, threadID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.threadIdx[threadID]
	count := 0
	for _, id := range ids {
		if s.items[id].Visible() {
			count++
		}
	}
	return count, nil
}

func (s *MemoryCommentStore) Close() error {
	if s.path == "" {
		return nil
	}
	s.mu.Lock()
	close(s.stop)
	s.mu.Unlock()
	<-s.done
	return s.flush()
}

func (s *MemoryCommentStore) saveLoop(interval time.Duration) {
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

func (s *MemoryCommentStore) flush() error {
	if s.path == "" {
		return nil
	}
	s.mu.Lock()
	snapshot := make([]*model.Comment, 0, len(s.items))
	for _, c := range s.items {
		snapshot = append(snapshot, c.Clone())
	}
	s.dirty = false
	s.mu.Unlock()
	sort.Slice(snapshot, func(i, j int) bool { return snapshot[i].CreatedAt.Before(snapshot[j].CreatedAt) })
	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal comments: %w", err)
	}
	dir := filepath.Dir(s.path)
	if dir != "" && dir != "." {
		os.MkdirAll(dir, 0o755) // #nosec G301
	}
	tmp, err := os.CreateTemp(dir, ".forumgo-comments-*.tmp")
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

func (s *MemoryCommentStore) load() error {
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
	var items []*model.Comment
	if err := json.Unmarshal(data, &items); err != nil {
		return fmt.Errorf("corrupt comment data: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range items {
		if c != nil && c.ID != "" {
			s.items[c.ID] = c
			s.threadIdx[c.ThreadID] = append(s.threadIdx[c.ThreadID], c.ID)
		}
	}
	return nil
}
