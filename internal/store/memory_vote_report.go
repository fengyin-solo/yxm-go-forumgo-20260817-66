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

// MemoryVoteStore is an in-memory vote store with JSON persistence.
type MemoryVoteStore struct {
	mu    sync.RWMutex
	items map[string]*model.Vote // key: targetType:targetID:userID
	path  string
	dirty bool
	log   *logger.Logger
	stop  chan struct{}
	done  chan struct{}
}

// NewMemoryVoteStore creates a MemoryVoteStore.
func NewMemoryVoteStore(path string, log *logger.Logger, interval time.Duration) (*MemoryVoteStore, error) {
	if log == nil {
		log = logger.Default()
	}
	s := &MemoryVoteStore{
		items: make(map[string]*model.Vote),
		path:  path,
		log:   log,
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
	if path != "" {
		if err := s.load(); err != nil {
			log.Warn("vote store load failed", "err", err.Error())
		}
		go s.saveLoop(interval)
	} else {
		close(s.done)
	}
	return s, nil
}

func voteKey(targetType, targetID, userID string) string {
	return targetType + ":" + targetID + ":" + userID
}

func (s *MemoryVoteStore) Upsert(ctx context.Context, v *model.Vote) error {
	if v == nil || v.ID == "" {
		return model.ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := voteKey(v.TargetType, v.TargetID, v.UserID)
	s.items[key] = v.Clone()
	s.dirty = true
	return nil
}

func (s *MemoryVoteStore) GetByTargetAndUser(ctx context.Context, targetType, targetID, userID string) (*model.Vote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[voteKey(targetType, targetID, userID)]
	if !ok {
		return nil, model.ErrNotFound
	}
	return v.Clone(), nil
}

func (s *MemoryVoteStore) CountByTarget(ctx context.Context, targetType, targetID string) (up, down int, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	prefix := targetType + ":" + targetID + ":"
	for _, v := range s.items {
		key := v.TargetType + ":" + v.TargetID + ":" + v.UserID
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			if v.Value > 0 {
				up++
			} else {
				down++
			}
		}
	}
	return up, down, nil
}

func (s *MemoryVoteStore) Close() error {
	if s.path == "" {
		return nil
	}
	s.mu.Lock()
	close(s.stop)
	s.mu.Unlock()
	<-s.done
	return s.flush()
}

func (s *MemoryVoteStore) saveLoop(interval time.Duration) {
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

func (s *MemoryVoteStore) flush() error {
	if s.path == "" {
		return nil
	}
	s.mu.Lock()
	snapshot := make([]*model.Vote, 0, len(s.items))
	for _, v := range s.items {
		snapshot = append(snapshot, v.Clone())
	}
	s.dirty = false
	s.mu.Unlock()
	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal votes: %w", err)
	}
	dir := filepath.Dir(s.path)
	if dir != "" && dir != "." {
		os.MkdirAll(dir, 0o755) // #nosec G301
	}
	tmp, err := os.CreateTemp(dir, ".forumgo-votes-*.tmp")
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

func (s *MemoryVoteStore) load() error {
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
	var items []*model.Vote
	if err := json.Unmarshal(data, &items); err != nil {
		return fmt.Errorf("corrupt vote data: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range items {
		if v != nil && v.ID != "" {
			s.items[voteKey(v.TargetType, v.TargetID, v.UserID)] = v
		}
	}
	return nil
}

// MemoryReportStore is an in-memory report store with JSON persistence.
type MemoryReportStore struct {
	mu    sync.RWMutex
	items map[string]*model.Report
	path  string
	dirty bool
	log   *logger.Logger
	stop  chan struct{}
	done  chan struct{}
}

// NewMemoryReportStore creates a MemoryReportStore.
func NewMemoryReportStore(path string, log *logger.Logger, interval time.Duration) (*MemoryReportStore, error) {
	if log == nil {
		log = logger.Default()
	}
	s := &MemoryReportStore{
		items: make(map[string]*model.Report),
		path:  path,
		log:   log,
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
	if path != "" {
		if err := s.load(); err != nil {
			log.Warn("report store load failed", "err", err.Error())
		}
		go s.saveLoop(interval)
	} else {
		close(s.done)
	}
	return s, nil
}

func (s *MemoryReportStore) Create(ctx context.Context, r *model.Report) error {
	if r == nil || r.ID == "" {
		return model.ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[r.ID] = r.Clone()
	s.dirty = true
	return nil
}

func (s *MemoryReportStore) GetByID(ctx context.Context, id string) (*model.Report, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.items[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return r.Clone(), nil
}

func (s *MemoryReportStore) Update(ctx context.Context, r *model.Report) error {
	if r == nil || r.ID == "" {
		return model.ErrInvalidInput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[r.ID]; !ok {
		return model.ErrNotFound
	}
	s.items[r.ID] = r.Clone()
	s.dirty = true
	return nil
}

func (s *MemoryReportStore) List(ctx context.Context, status string) ([]*model.Report, error) {
	s.mu.RLock()
	var out []*model.Report
	for _, r := range s.items {
		if status == "" || r.Status == status {
			out = append(out, r.Clone())
		}
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *MemoryReportStore) Close() error {
	if s.path == "" {
		return nil
	}
	s.mu.Lock()
	close(s.stop)
	s.mu.Unlock()
	<-s.done
	return s.flush()
}

func (s *MemoryReportStore) saveLoop(interval time.Duration) {
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

func (s *MemoryReportStore) flush() error {
	if s.path == "" {
		return nil
	}
	s.mu.Lock()
	snapshot := make([]*model.Report, 0, len(s.items))
	for _, r := range s.items {
		snapshot = append(snapshot, r.Clone())
	}
	s.dirty = false
	s.mu.Unlock()
	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal reports: %w", err)
	}
	dir := filepath.Dir(s.path)
	if dir != "" && dir != "." {
		os.MkdirAll(dir, 0o755) // #nosec G301
	}
	tmp, err := os.CreateTemp(dir, ".forumgo-reports-*.tmp")
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

func (s *MemoryReportStore) load() error {
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
	var items []*model.Report
	if err := json.Unmarshal(data, &items); err != nil {
		return fmt.Errorf("corrupt report data: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range items {
		if r != nil && r.ID != "" {
			s.items[r.ID] = r
		}
	}
	return nil
}
