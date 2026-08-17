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

// MemoryThreadStore is an in-memory thread store with JSON persistence.
type MemoryThreadStore struct {
	mu      sync.RWMutex
	items   map[string]*model.Thread
	boardIdx map[string][]string // boardID -> thread IDs
	path    string
	dirty   bool
	log     *logger.Logger
	stop    chan struct{}
	done    chan struct{}
}

// NewMemoryThreadStore creates a MemoryThreadStore.
func NewMemoryThreadStore(path string, log *logger.Logger, interval time.Duration) (*MemoryThreadStore, error) {
	if log == nil {
		log = logger.Default()
	}
	s := &MemoryThreadStore{
		items:    make(map[string]*model.Thread),
		boardIdx: make(map[string][]string),
		path:     path,
		log:      log,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
	if path != "" {
		if err := s.load(); err != nil {
			log.Warn("thread store load failed", "err", err.Error())
		}
		go s.saveLoop(interval)
	} else {
		close(s.done)
	}
	return s, nil
}

func (s *MemoryThreadStore) Create(ctx context.Context, t *model.Thread) error {
	if t == nil || t.ID == "" {
		return model.ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.items[t.ID]; exists {
		return model.ErrAlreadyExists
	}
	s.items[t.ID] = t.Clone()
	s.boardIdx[t.BoardID] = append(s.boardIdx[t.BoardID], t.ID)
	s.dirty = true
	return nil
}

func (s *MemoryThreadStore) GetByID(ctx context.Context, id string) (*model.Thread, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.items[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return t.Clone(), nil
}

func (s *MemoryThreadStore) Update(ctx context.Context, t *model.Thread) error {
	if t == nil || t.ID == "" {
		return model.ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[t.ID]; !ok {
		return model.ErrNotFound
	}
	s.items[t.ID] = t.Clone()
	s.dirty = true
	return nil
}

func (s *MemoryThreadStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.items[id]
	if !ok {
		return model.ErrNotFound
	}
	// Remove from board index
	ids := s.boardIdx[t.BoardID]
	for i, tid := range ids {
		if tid == id {
			s.boardIdx[t.BoardID] = append(ids[:i], ids[i+1:]...)
			break
		}
	}
	delete(s.items, id)
	s.dirty = true
	return nil
}

func (s *MemoryThreadStore) List(ctx context.Context, f model.ThreadFilter) ([]*model.Thread, error) {
	f.Normalize()
	s.mu.RLock()
	var ids []string
	if f.BoardID != "" {
		ids = s.boardIdx[f.BoardID]
	} else {
		ids = make([]string, 0, len(s.items))
		for id := range s.items {
			ids = append(ids, id)
		}
	}
	var out []*model.Thread
	for _, id := range ids {
		t := s.items[id]
		if f.Matches(t) {
			out = append(out, t.Clone())
		}
	}
	s.mu.RUnlock()
	// Sort: pinned first, then by last_reply_at descending (null = use created_at)
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsPinned != out[j].IsPinned {
			return out[i].IsPinned
		}
		iTime := out[i].LastReplyAt
		if iTime == nil {
			iTime = &out[i].CreatedAt
		}
		jTime := out[j].LastReplyAt
		if jTime == nil {
			jTime = &out[j].CreatedAt
		}
		return iTime.After(*jTime)
	})
	return out, nil
}

func (s *MemoryThreadStore) IncrementViewCount(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.items[id]
	if !ok {
		return model.ErrNotFound
	}
	t.ViewCount++
	s.dirty = true
	return nil
}

func (s *MemoryThreadStore) Close() error {
	if s.path == "" {
		return nil
	}
	s.mu.Lock()
	close(s.stop)
	s.mu.Unlock()
	<-s.done
	return s.flush()
}

func (s *MemoryThreadStore) saveLoop(interval time.Duration) {
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

func (s *MemoryThreadStore) flush() error {
	if s.path == "" {
		return nil
	}
	s.mu.Lock()
	snapshot := make([]*model.Thread, 0, len(s.items))
	for _, t := range s.items {
		snapshot = append(snapshot, t.Clone())
	}
	s.dirty = false
	s.mu.Unlock()
	sort.Slice(snapshot, func(i, j int) bool {
		if snapshot[i].IsPinned != snapshot[j].IsPinned {
			return snapshot[i].IsPinned
		}
		return snapshot[i].CreatedAt.Before(snapshot[j].CreatedAt)
	})
	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal threads: %w", err)
	}
	dir := filepath.Dir(s.path)
	if dir != "" && dir != "." {
		os.MkdirAll(dir, 0o755) // #nosec G301
	}
	tmp, err := os.CreateTemp(dir, ".forumgo-threads-*.tmp")
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

func (s *MemoryThreadStore) load() error {
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
	var items []*model.Thread
	if err := json.Unmarshal(data, &items); err != nil {
		return fmt.Errorf("corrupt thread data: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range items {
		if t != nil && t.ID != "" {
			s.items[t.ID] = t
			s.boardIdx[t.BoardID] = append(s.boardIdx[t.BoardID], t.ID)
		}
	}
	return nil
}
