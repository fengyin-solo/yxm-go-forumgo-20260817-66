package store

import (
	"context"
	"testing"
	"time"

	"github.com/example/forumgo/internal/model"
)

func TestBoardStore(t *testing.T) {
	s, _ := NewMemoryBoardStore("", nil, 30*time.Second)
	defer s.Close()

	b := &model.Board{ID: "b1", Name: "  General  ", Slug: "general", DisplayOrder: 1}
	if err := s.Create(context.Background(), b); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.GetByID(context.Background(), "b1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "  General  " {
		t.Errorf("Name = %q (not normalized by store)", got.Name)
	}
	// Clone independence
	got.Name = "Changed"
	if orig, _ := s.GetByID(context.Background(), "b1"); orig.Name == "Changed" {
		t.Error("clone leaked name")
	}
	// Filter
	list, _ := s.List(context.Background(), model.BoardFilter{Query: "general"})
	if len(list) != 1 {
		t.Errorf("List len = %d", len(list))
	}
	// Delete
	if err := s.Delete(context.Background(), "b1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.GetByID(context.Background(), "b1"); err != model.ErrNotFound {
		t.Errorf("after delete: err = %v", err)
	}
}

func TestThreadStore(t *testing.T) {
	s, _ := NewMemoryThreadStore("", nil, 30*time.Second)
	defer s.Close()

	t1 := &model.Thread{ID: "t1", BoardID: "b1", AuthorID: "u1", Title: "Hello", Body: "World"}
	if err := s.Create(context.Background(), t1); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.GetByID(context.Background(), "t1")
	if err != nil || got.Title != "Hello" {
		t.Errorf("GetByID: err=%v, title=%q", err, got.Title)
	}
	// Filter by board
	list, _ := s.List(context.Background(), model.ThreadFilter{BoardID: "b1"})
	if len(list) != 1 {
		t.Errorf("List len = %d", len(list))
	}
	// View count
	s.IncrementViewCount(context.Background(), "t1")
	got, _ = s.GetByID(context.Background(), "t1")
	if got.ViewCount != 1 {
		t.Errorf("ViewCount = %d", got.ViewCount)
	}
}

func TestCommentStore(t *testing.T) {
	s, _ := NewMemoryCommentStore("", nil, 30*time.Second)
	defer s.Close()

	c := &model.Comment{ID: "c1", ThreadID: "t1", AuthorID: "u1", Body: "A comment"}
	if err := s.Create(context.Background(), c); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.GetByID(context.Background(), "c1")
	if err != nil || got.Body != "A comment" {
		t.Errorf("GetByID: err=%v, body=%q", err, got.Body)
	}
	// Soft delete
	s.SoftDelete(context.Background(), "c1")
	got, _ = s.GetByID(context.Background(), "c1")
	if !got.IsDeleted {
		t.Error("IsDeleted should be true")
	}
}

func TestVoteStore(t *testing.T) {
	s, _ := NewMemoryVoteStore("", nil, 30*time.Second)
	defer s.Close()

	v := &model.Vote{ID: "v1", TargetType: "thread", TargetID: "t1", UserID: "u1", Value: 1}
	if err := s.Upsert(context.Background(), v); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	got, err := s.GetByTargetAndUser(context.Background(), "thread", "t1", "u1")
	if err != nil || got.Value != 1 {
		t.Errorf("GetByTargetAndUser: err=%v, value=%d", err, got.Value)
	}
	up, down, _ := s.CountByTarget(context.Background(), "thread", "t1")
	if up != 1 || down != 0 {
		t.Errorf("CountByTarget: up=%d, down=%d", up, down)
	}
}

func TestReportStore(t *testing.T) {
	s, _ := NewMemoryReportStore("", nil, 30*time.Second)
	defer s.Close()

	r := &model.Report{ID: "r1", ReporterID: "u1", TargetType: "thread", TargetID: "t1", Reason: "spam", Status: "open"}
	if err := s.Create(context.Background(), r); err != nil {
		t.Fatalf("Create: %v", err)
	}
	list, _ := s.List(context.Background(), "open")
	if len(list) != 1 {
		t.Errorf("List len = %d", len(list))
	}
	// List all (no filter)
	all, _ := s.List(context.Background(), "")
	if len(all) != 1 {
		t.Errorf("List all len = %d", len(all))
	}
}

func ptrBool(b bool) *bool { return &b }
