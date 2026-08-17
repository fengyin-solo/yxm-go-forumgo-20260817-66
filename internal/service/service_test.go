package service

import (
	"context"
	"testing"
	"time"

	"github.com/example/forumgo/internal/model"
	"github.com/example/forumgo/internal/store"
)

func boardTest(t *testing.T) *BoardService {
	boardStore, _ := store.NewMemoryBoardStore("", nil, 30*time.Second)
	t.Cleanup(func() { boardStore.Close() })
	return NewBoardService(boardStore)
}

func threadTest(t *testing.T) (*BoardService, *ThreadService) {
	boardStore, _ := store.NewMemoryBoardStore("", nil, 30*time.Second)
	threadStore, _ := store.NewMemoryThreadStore("", nil, 30*time.Second)
	t.Cleanup(func() {
		boardStore.Close()
		threadStore.Close()
	})
	return NewBoardService(boardStore), NewThreadService(threadStore, boardStore)
}

func commentTest(t *testing.T) (*BoardService, *ThreadService, *CommentService) {
	boardStore, _ := store.NewMemoryBoardStore("", nil, 30*time.Second)
	threadStore, _ := store.NewMemoryThreadStore("", nil, 30*time.Second)
	commentStore, _ := store.NewMemoryCommentStore("", nil, 30*time.Second)
	t.Cleanup(func() {
		boardStore.Close()
		threadStore.Close()
		commentStore.Close()
	})
	return NewBoardService(boardStore),
		NewThreadService(threadStore, boardStore),
		NewCommentService(commentStore, threadStore)
}

func TestBoardCreate(t *testing.T) {
	svc := boardTest(t)
	b, err := svc.Create(context.Background(), &model.Board{Name: "  General  ", Slug: "general"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if b.Name != "General" {
		t.Errorf("Name = %q (should be normalized)", b.Name)
	}
	if b.ID == "" {
		t.Error("ID should be generated")
	}
}

func TestBoardCreateEmptyName(t *testing.T) {
	svc := boardTest(t)
	if _, err := svc.Create(context.Background(), &model.Board{Name: ""}); err == nil {
		t.Error("empty name should error")
	}
}

func TestBoardAutoSlug(t *testing.T) {
	svc := boardTest(t)
	b, _ := svc.Create(context.Background(), &model.Board{Name: "Hello World"})
	if b.Slug != "hello-world" {
		t.Errorf("Slug = %q", b.Slug)
	}
}

func TestThreadCreate(t *testing.T) {
	boardSvc, threadSvc := threadTest(t)
	boardSvc.Create(context.Background(), &model.Board{ID: "b1", Name: "Board", Slug: "board"})
	t1, err := threadSvc.Create(context.Background(), &model.Thread{BoardID: "b1", AuthorID: "u1", Title: "Hello", Body: "World"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if t1.ID == "" {
		t.Error("ID should be generated")
	}
	if t1.ReplyCount != 0 {
		t.Errorf("ReplyCount = %d", t1.ReplyCount)
	}
}

func TestThreadCreateLockedBoard(t *testing.T) {
	boardSvc, threadSvc := threadTest(t)
	b, _ := boardSvc.Create(context.Background(), &model.Board{ID: "b1", Name: "Board", Slug: "board"})
	boardSvc.Update(context.Background(), b.ID, &UpdateBoardRequest{IsLocked: ptrBool(true)})
	if _, err := threadSvc.Create(context.Background(), &model.Thread{BoardID: "b1", AuthorID: "u1", Title: "Hello", Body: "World"}); err == nil {
		t.Error("should error on locked board")
	}
}

func TestCommentCreate(t *testing.T) {
	boardSvc, threadSvc, commentSvc := commentTest(t)
	boardSvc.Create(context.Background(), &model.Board{ID: "b1", Name: "Board", Slug: "board"})
	t1, _ := threadSvc.Create(context.Background(), &model.Thread{BoardID: "b1", AuthorID: "u1", Title: "Hello", Body: "World"})
	c1, err := commentSvc.Create(context.Background(), &model.Comment{ThreadID: t1.ID, AuthorID: "u1", Body: "Nice!"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c1.ID == "" {
		t.Error("ID should be generated")
	}
	// Check thread reply count updated
	t1, _ = threadSvc.GetByID(context.Background(), t1.ID)
	if t1.ReplyCount != 1 {
		t.Errorf("ReplyCount = %d", t1.ReplyCount)
	}
}

func TestCommentLockedThread(t *testing.T) {
	boardSvc, threadSvc, commentSvc := commentTest(t)
	boardSvc.Create(context.Background(), &model.Board{ID: "b1", Name: "Board", Slug: "board"})
	t1, _ := threadSvc.Create(context.Background(), &model.Thread{BoardID: "b1", AuthorID: "u1", Title: "Hello", Body: "World"})
	threadSvc.Update(context.Background(), t1.ID, &UpdateThreadRequest{IsLocked: ptrBool(true)})
	if _, err := commentSvc.Create(context.Background(), &model.Comment{ThreadID: t1.ID, AuthorID: "u1", Body: "Reply"}); err == nil {
		t.Error("should error on locked thread")
	}
}

func TestCommentUpdate(t *testing.T) {
	boardSvc, threadSvc, commentSvc := commentTest(t)
	boardSvc.Create(context.Background(), &model.Board{ID: "b1", Name: "Board", Slug: "board"})
	t1, _ := threadSvc.Create(context.Background(), &model.Thread{BoardID: "b1", AuthorID: "u1", Title: "Hello", Body: "World"})
	c1, _ := commentSvc.Create(context.Background(), &model.Comment{ThreadID: t1.ID, AuthorID: "u1", Body: "A"})
	c2, err := commentSvc.Update(context.Background(), c1.ID, &UpdateCommentRequest{Body: ptrStr("B")})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if c2.Body != "B" {
		t.Errorf("Body = %q", c2.Body)
	}
}

func TestVoteService(t *testing.T) {
	voteStore, _ := store.NewMemoryVoteStore("", nil, 30*time.Second)
	defer voteStore.Close()
	voteSvc := NewVoteService(voteStore, nil, nil)

	// Invalid vote value
	v := &model.Vote{TargetType: "thread", TargetID: "t1", UserID: "u1", Value: 2}
	if err := voteSvc.Vote(context.Background(), v); err == nil {
		t.Error("invalid vote value should error")
	}
	// Wrong target type
	v2 := &model.Vote{TargetType: "invalid", TargetID: "t1", UserID: "u1", Value: 1}
	if err := voteSvc.Vote(context.Background(), v2); err == nil {
		t.Error("wrong target type should error")
	}
	// Count with no votes
	up, down, _ := voteSvc.GetVotes(context.Background(), "thread", "t1")
	if up != 0 || down != 0 {
		t.Errorf("GetVotes: up=%d, down=%d", up, down)
	}
}

func TestSlugify(t *testing.T) {
	cases := []struct {
		in, out string
	}{
		{"Hello World", "hello-world"},
		{"  Spaces  ", "spaces"},
		{"A--B", "a-b"},
		{"Already-lower", "already-lower"},
		{"Special!@#$%Chars", "specialchars"},
	}
	for _, c := range cases {
		if got := slugify(c.in); got != c.out {
			t.Errorf("slugify(%q) = %q, want %q", c.in, got, c.out)
		}
	}
}

func ptrBool(b bool) *bool   { return &b }
func ptrStr(s string) *string { return &s }
