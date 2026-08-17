// Package model defines the domain entities for the forumgo service.
package model

import (
	"strings"
	"time"
)

const (
	TargetThread    = "thread"
	TargetComment   = "comment"
	ReportOpen      = "open"
	ReportResolved  = "resolved"
	ReportDismissed = "dismissed"
)

// CanonicalReportStatus returns the normalized moderation status.
func CanonicalReportStatus(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

// Board represents a forum category/board.
type Board struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"` // URL-friendly name, unique
	Description  string    `json:"description,omitempty"`
	DisplayOrder int       `json:"display_order"`
	IsLocked     bool      `json:"is_locked"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Clone returns a deep copy of the board.
func (b *Board) Clone() *Board {
	if b == nil {
		return nil
	}
	cp := *b
	return &cp
}

// Normalize trims whitespace and lowercases the slug.
func (b *Board) Normalize() {
	b.Name = strings.TrimSpace(b.Name)
	b.Slug = strings.ToLower(strings.TrimSpace(b.Slug))
	b.Description = strings.TrimSpace(b.Description)
}

// Thread represents a discussion thread in a board.
type Thread struct {
	ID         string     `json:"id"`
	BoardID    string     `json:"board_id"`
	AuthorID   string     `json:"author_id"`
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	IsPinned   bool       `json:"is_pinned"`
	IsLocked   bool       `json:"is_locked"`
	ViewCount  int        `json:"view_count"`
	ReplyCount int        `json:"reply_count"`
	LastReplyAt *time.Time `json:"last_reply_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// Clone returns a deep copy of the thread.
func (t *Thread) Clone() *Thread {
	if t == nil {
		return nil
	}
	cp := *t
	if t.LastReplyAt != nil {
		lt := t.LastReplyAt.In(time.UTC)
		cp.LastReplyAt = &lt
	}
	return &cp
}

// Normalize trims whitespace from title and body.
func (t *Thread) Normalize() {
	t.Title = strings.TrimSpace(t.Title)
	t.Body = strings.TrimSpace(t.Body)
}

// IncrementReply increments reply count and updates last reply time.
func (t *Thread) IncrementReply(at time.Time) {
	t.ReplyCount++
	t.LastReplyAt = &at
}

// Comment represents a reply in a thread.
type Comment struct {
	ID        string     `json:"id"`
	ThreadID  string     `json:"thread_id"`
	ParentID  *string    `json:"parent_id,omitempty"` // nil for top-level replies
	AuthorID  string     `json:"author_id"`
	Body      string     `json:"body"`
	Upvotes   int        `json:"upvotes"`
	Downvotes int        `json:"downvotes"`
	IsDeleted bool       `json:"is_deleted"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Clone returns a deep copy of the comment.
func (c *Comment) Clone() *Comment {
	if c == nil {
		return nil
	}
	cp := *c
	if c.ParentID != nil {
		pid := *c.ParentID
		cp.ParentID = &pid
	}
	return &cp
}

// Normalize trims whitespace from body.
func (c *Comment) Normalize() {
	c.Body = strings.TrimSpace(c.Body)
}

// Score returns the net vote score.
func (c *Comment) Score() int {
	return c.Upvotes - c.Downvotes
}

// Vote records a user's vote on a thread or comment.
type Vote struct {
	ID        string    `json:"id"`
	TargetType string   `json:"target_type"` // "thread" or "comment"
	TargetID  string    `json:"target_id"`
	UserID    string    `json:"user_id"`
	Value     int       `json:"value"` // +1 or -1
	CreatedAt time.Time `json:"created_at"`
}

// Clone returns a deep copy of the vote.
func (v *Vote) Clone() *Vote {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

// Report records a user-submitted report.
type Report struct {
	ID          string     `json:"id"`
	ReporterID  string     `json:"reporter_id"`
	TargetType  string     `json:"target_type"` // "thread" or "comment"
	TargetID    string     `json:"target_id"`
	Reason      string     `json:"reason"`
	Status      string     `json:"status"` // "open", "resolved", "dismissed"
	CreatedAt   time.Time  `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// Clone returns a deep copy of the report.
func (r *Report) Clone() *Report {
	if r == nil {
		return nil
	}
	cp := *r
	if r.ResolvedAt != nil {
		rt := r.ResolvedAt.In(time.UTC)
		cp.ResolvedAt = &rt
	}
	return &cp
}

// BoardFilter constrains board listing queries.
type BoardFilter struct {
	Query string
}

// Matches reports whether the board satisfies the filter.
func (f *BoardFilter) Matches(b *Board) bool {
	if f.Query == "" {
		return true
	}
	q := strings.ToLower(f.Query)
	return strings.Contains(strings.ToLower(b.Name), q) ||
		strings.Contains(strings.ToLower(b.Slug), q) ||
		strings.Contains(strings.ToLower(b.Description), q)
}

// ThreadFilter constrains thread listing queries.
type ThreadFilter struct {
	BoardID    string
	AuthorID   string
	Query      string
	IsPinned   *bool
	IsLocked   *bool
}

// Matches reports whether the thread satisfies the filter.
func (f *ThreadFilter) Matches(t *Thread) bool {
	if f.BoardID != "" && t.BoardID != f.BoardID {
		return false
	}
	if f.AuthorID != "" && t.AuthorID != f.AuthorID {
		return false
	}
	if f.IsPinned != nil && t.IsPinned != *f.IsPinned {
		return false
	}
	if f.IsLocked != nil && t.IsLocked != *f.IsLocked {
		return false
	}
	if f.Query != "" {
		q := strings.ToLower(f.Query)
		return strings.Contains(strings.ToLower(t.Title), q) ||
			strings.Contains(strings.ToLower(t.Body), q)
	}
	return true
}

// CommentFilter constrains comment listing queries.
type CommentFilter struct {
	ThreadID  string
	ParentID  *bool // nil = all, true = only top-level, false = only replies
	AuthorID  string
	SortBy    string // "newest", "oldest", "score"
}
