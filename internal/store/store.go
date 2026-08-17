package store

import (
	"context"

	"github.com/example/forumgo/internal/model"
)

// BoardStore manages boards.
type BoardStore interface {
	Create(ctx context.Context, b *model.Board) error
	GetByID(ctx context.Context, id string) (*model.Board, error)
	GetBySlug(ctx context.Context, slug string) (*model.Board, error)
	Update(ctx context.Context, b *model.Board) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, f model.BoardFilter) ([]*model.Board, error)
}

// ThreadStore manages threads.
type ThreadStore interface {
	Create(ctx context.Context, t *model.Thread) error
	GetByID(ctx context.Context, id string) (*model.Thread, error)
	Update(ctx context.Context, t *model.Thread) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, f model.ThreadFilter) ([]*model.Thread, error)
	IncrementViewCount(ctx context.Context, id string) error
}

// CommentStore manages comments.
type CommentStore interface {
	Create(ctx context.Context, c *model.Comment) error
	GetByID(ctx context.Context, id string) (*model.Comment, error)
	Update(ctx context.Context, c *model.Comment) error
	SoftDelete(ctx context.Context, id string) error
	List(ctx context.Context, f model.CommentFilter) ([]*model.Comment, error)
	CountByThread(ctx context.Context, threadID string) (int, error)
}

// VoteStore manages votes.
type VoteStore interface {
	Upsert(ctx context.Context, v *model.Vote) error
	GetByTargetAndUser(ctx context.Context, targetType, targetID, userID string) (*model.Vote, error)
	CountByTarget(ctx context.Context, targetType, targetID string) (up, down int, _ error)
}

// ReportStore manages reports.
type ReportStore interface {
	Create(ctx context.Context, r *model.Report) error
	GetByID(ctx context.Context, id string) (*model.Report, error)
	Update(ctx context.Context, r *model.Report) error
	List(ctx context.Context, status string) ([]*model.Report, error)
}
