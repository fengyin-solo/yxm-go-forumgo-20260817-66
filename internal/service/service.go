package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/example/forumgo/internal/model"
	"github.com/example/forumgo/internal/store"
)

// BoardService handles board business logic.
type BoardService struct {
	store store.BoardStore
	now   func() time.Time
}

// NewBoardService creates a new BoardService.
func NewBoardService(store store.BoardStore) *BoardService {
	return &BoardService{store: store, now: time.Now}
}

// Create creates a new board.
func (s *BoardService) Create(ctx context.Context, b *model.Board) (*model.Board, error) {
	b.Normalize()
	if b.Name == "" {
		return nil, fmt.Errorf("%w: name is required", model.ErrInvalidInput)
	}
	if b.Slug == "" {
		b.Slug = slugify(b.Name)
	}
	if b.ID == "" {
		b.ID = newID()
	}
	b.CreatedAt = s.now().UTC()
	b.UpdatedAt = b.CreatedAt
	if err := s.store.Create(ctx, b); err != nil {
		return nil, err
	}
	return b.Clone(), nil
}

// GetByID returns a board by ID.
func (s *BoardService) GetByID(ctx context.Context, id string) (*model.Board, error) {
	return s.store.GetByID(ctx, id)
}

// GetBySlug returns a board by slug.
func (s *BoardService) GetBySlug(ctx context.Context, slug string) (*model.Board, error) {
	return s.store.GetBySlug(ctx, slug)
}

// Update updates an existing board.
func (s *BoardService) Update(ctx context.Context, id string, req *UpdateBoardRequest) (*model.Board, error) {
	b, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	req.Apply(b)
	b.Normalize()
	if b.Name == "" {
		return nil, fmt.Errorf("%w: name is required", model.ErrInvalidInput)
	}
	b.UpdatedAt = s.now().UTC()
	if err := s.store.Update(ctx, b); err != nil {
		return nil, err
	}
	return b.Clone(), nil
}

// Delete deletes a board.
func (s *BoardService) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// List returns boards matching the filter.
func (s *BoardService) List(ctx context.Context, f model.BoardFilter) ([]*model.Board, error) {
	return s.store.List(ctx, f)
}

// UpdateBoardRequest describes fields that can be updated on a board.
type UpdateBoardRequest struct {
	Name         *string `json:"name"`
	Slug         *string `json:"slug"`
	Description  *string `json:"description"`
	DisplayOrder *int    `json:"display_order"`
	IsLocked     *bool   `json:"is_locked"`
}

func (r *UpdateBoardRequest) Apply(b *model.Board) {
	if r.Name != nil {
		b.Name = *r.Name
	}
	if r.Slug != nil {
		b.Slug = *r.Slug
	}
	if r.Description != nil {
		b.Description = *r.Description
	}
	if r.DisplayOrder != nil {
		b.DisplayOrder = *r.DisplayOrder
	}
	if r.IsLocked != nil {
		b.IsLocked = *r.IsLocked
	}
}

// ThreadService handles thread business logic.
type ThreadService struct {
	store     store.ThreadStore
	boardStore store.BoardStore
	now       func() time.Time
}

// NewThreadService creates a new ThreadService.
func NewThreadService(store store.ThreadStore, boardStore store.BoardStore) *ThreadService {
	return &ThreadService{store: store, boardStore: boardStore, now: time.Now}
}

// Create creates a new thread.
func (s *ThreadService) Create(ctx context.Context, t *model.Thread) (*model.Thread, error) {
	t.Normalize()
	if t.Title == "" {
		return nil, fmt.Errorf("%w: title is required", model.ErrInvalidInput)
	}
	if t.BoardID == "" {
		return nil, fmt.Errorf("%w: board_id is required", model.ErrInvalidInput)
	}
	// Check board exists and is not locked
	board, err := s.boardStore.GetByID(ctx, t.BoardID)
	if err != nil {
		return nil, err
	}
	if board.IsLocked {
		return nil, fmt.Errorf("%w: board is locked", model.ErrConflict)
	}
	if t.ID == "" {
		t.ID = newID()
	}
	t.ViewCount = 0
	t.ReplyCount = 0
	t.CreatedAt = s.now().UTC()
	t.UpdatedAt = t.CreatedAt
	if err := s.store.Create(ctx, t); err != nil {
		return nil, err
	}
	return t.Clone(), nil
}

// GetByID returns a thread by ID and increments view count.
func (s *ThreadService) GetByID(ctx context.Context, id string) (*model.Thread, error) {
	t, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// Increment view count (best-effort)
	s.store.IncrementViewCount(ctx, id)
	return t, nil
}

// Update updates an existing thread.
func (s *ThreadService) Update(ctx context.Context, id string, req *UpdateThreadRequest) (*model.Thread, error) {
	t, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	req.Apply(t)
	t.Normalize()
	if t.Title == "" {
		return nil, fmt.Errorf("%w: title is required", model.ErrInvalidInput)
	}
	t.UpdatedAt = s.now().UTC()
	if err := s.store.Update(ctx, t); err != nil {
		return nil, err
	}
	return t.Clone(), nil
}

// Delete deletes a thread.
func (s *ThreadService) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// List returns threads matching the filter.
func (s *ThreadService) List(ctx context.Context, f model.ThreadFilter) ([]*model.Thread, error) {
	return s.store.List(ctx, f)
}

// UpdateThreadRequest describes fields that can be updated on a thread.
type UpdateThreadRequest struct {
	Title    *string `json:"title"`
	Body     *string `json:"body"`
	IsPinned *bool   `json:"is_pinned"`
	IsLocked *bool   `json:"is_locked"`
}

func (r *UpdateThreadRequest) Apply(t *model.Thread) {
	if r.Title != nil {
		t.Title = *r.Title
	}
	if r.Body != nil {
		t.Body = *r.Body
	}
	if r.IsPinned != nil {
		t.IsPinned = *r.IsPinned
	}
	if r.IsLocked != nil {
		t.IsLocked = *r.IsLocked
	}
}

// CommentService handles comment business logic.
type CommentService struct {
	store     store.CommentStore
	threadStore store.ThreadStore
	now       func() time.Time
}

// NewCommentService creates a new CommentService.
func NewCommentService(store store.CommentStore, threadStore store.ThreadStore) *CommentService {
	return &CommentService{store: store, threadStore: threadStore, now: time.Now}
}

// Create creates a new comment.
func (s *CommentService) Create(ctx context.Context, c *model.Comment) (*model.Comment, error) {
	c.Normalize()
	if c.Body == "" {
		return nil, fmt.Errorf("%w: body is required", model.ErrInvalidInput)
	}
	if c.ThreadID == "" {
		return nil, fmt.Errorf("%w: thread_id is required", model.ErrInvalidInput)
	}
	// Check thread exists
	t, err := s.threadStore.GetByID(ctx, c.ThreadID)
	if err != nil {
		return nil, err
	}
	if t.IsLocked {
		return nil, fmt.Errorf("%w: thread is locked", model.ErrConflict)
	}
	c.ID = newID()
	c.Upvotes = 0
	c.Downvotes = 0
	c.IsDeleted = false
	c.CreatedAt = s.now().UTC()
	c.UpdatedAt = c.CreatedAt
	if err := s.store.Create(ctx, c); err != nil {
		return nil, err
	}
	// Update thread reply count and last_reply_at
	now := s.now().UTC()
	t.IncrementReply(now)
	t.UpdatedAt = now
	s.threadStore.Update(ctx, t)
	return c.Clone(), nil
}

// GetByID returns a comment by ID.
func (s *CommentService) GetByID(ctx context.Context, id string) (*model.Comment, error) {
	return s.store.GetByID(ctx, id)
}

// Update updates an existing comment.
func (s *CommentService) Update(ctx context.Context, id string, req *UpdateCommentRequest) (*model.Comment, error) {
	c, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.IsDeleted {
		return nil, fmt.Errorf("%w: comment is deleted", model.ErrConflict)
	}
	req.Apply(c)
	c.Normalize()
	if c.Body == "" {
		return nil, fmt.Errorf("%w: body is required", model.ErrInvalidInput)
	}
	c.UpdatedAt = s.now().UTC()
	if err := s.store.Update(ctx, c); err != nil {
		return nil, err
	}
	return c.Clone(), nil
}

// Delete soft-deletes a comment.
func (s *CommentService) Delete(ctx context.Context, id string) error {
	return s.store.SoftDelete(ctx, id)
}

// List returns comments matching the filter.
func (s *CommentService) List(ctx context.Context, f model.CommentFilter) ([]*model.Comment, error) {
	return s.store.List(ctx, f)
}

// UpdateCommentRequest describes fields that can be updated on a comment.
type UpdateCommentRequest struct {
	Body *string `json:"body"`
}

func (r *UpdateCommentRequest) Apply(c *model.Comment) {
	if r.Body != nil {
		c.Body = *r.Body
	}
}

// VoteService handles vote business logic.
type VoteService struct {
	store       store.VoteStore
	threadStore store.ThreadStore
	commentStore store.CommentStore
	now         func() time.Time
}

// NewVoteService creates a new VoteService.
func NewVoteService(store store.VoteStore, threadStore store.ThreadStore, commentStore store.CommentStore) *VoteService {
	return &VoteService{store: store, threadStore: threadStore, commentStore: commentStore, now: time.Now}
}

// Vote casts or updates a vote.
func (s *VoteService) Vote(ctx context.Context, v *model.Vote) error {
	if v.Value != 1 && v.Value != -1 {
		return fmt.Errorf("%w: vote value must be +1 or -1", model.ErrInvalidInput)
	}
	if v.TargetType != "thread" && v.TargetType != "comment" {
		return fmt.Errorf("%w: target_type must be thread or comment", model.ErrInvalidInput)
	}
	if v.TargetID == "" || v.UserID == "" {
		return fmt.Errorf("%w: target_id and user_id required", model.ErrInvalidInput)
	}
	// Check target exists
	switch v.TargetType {
	case "thread":
		if _, err := s.threadStore.GetByID(ctx, v.TargetID); err != nil {
			return err
		}
	case "comment":
		if _, err := s.commentStore.GetByID(ctx, v.TargetID); err != nil {
			return err
		}
	}
	v.ID = newID()
	v.CreatedAt = s.now().UTC()
	return s.store.Upsert(ctx, v)
}

// GetVotes returns vote counts for a target.
func (s *VoteService) GetVotes(ctx context.Context, targetType, targetID string) (up, down int, err error) {
	return s.store.CountByTarget(ctx, targetType, targetID)
}

// ReportService handles report business logic.
type ReportService struct {
	store store.ReportStore
	now   func() time.Time
}

// NewReportService creates a new ReportService.
func NewReportService(store store.ReportStore) *ReportService {
	return &ReportService{store: store, now: time.Now}
}

// Create creates a new report.
func (s *ReportService) Create(ctx context.Context, r *model.Report) (*model.Report, error) {
	r.Reason = strings.TrimSpace(r.Reason)
	r.TargetType = strings.TrimSpace(r.TargetType)
	r.TargetID = strings.TrimSpace(r.TargetID)
	if r.Reason == "" {
		return nil, fmt.Errorf("%w: reason is required", model.ErrInvalidInput)
	}
	if r.TargetID == "" || r.TargetType == "" {
		return nil, fmt.Errorf("%w: target_id and target_type required", model.ErrInvalidInput)
	}
	r.ID = newID()
	r.Status = model.ReportOpen
	r.CreatedAt = s.now().UTC()
	if err := s.store.Create(ctx, r); err != nil {
		return nil, err
	}
	return r.Clone(), nil
}

// Resolve resolves or dismisses a report.
func (s *ReportService) Resolve(ctx context.Context, id string, status string) (*model.Report, error) {
	r, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	status = model.CanonicalReportStatus(status)
	if status != model.ReportResolved && status != model.ReportDismissed {
		return nil, fmt.Errorf("%w: status must be resolved or dismissed", model.ErrInvalidInput)
	}
	r.Status = status
	now := s.now().UTC()
	r.ResolvedAt = &now
	if err := s.store.Update(ctx, r); err != nil {
		return nil, err
	}
	return r.Clone(), nil
}

// List returns reports matching the filter.
func (s *ReportService) List(ctx context.Context, status string) ([]*model.Report, error) {
	return s.store.List(ctx, model.CanonicalReportStatus(status))
}

func newID() string {
	buf := make([]byte, 16)
	rand.Read(buf) // #nosec G404
	return fmt.Sprintf("%x", buf)
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b = append(b, c)
		} else if c == ' ' || c == '-' || c == '_' {
			if len(b) == 0 || b[len(b)-1] != '-' {
				b = append(b, '-')
			}
		}
	}
	if len(b) > 0 && b[len(b)-1] == '-' {
		b = b[:len(b)-1]
	}
	return string(b)
}
