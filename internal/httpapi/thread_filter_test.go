package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/forumgo/internal/model"
	"github.com/example/forumgo/internal/service"
	"github.com/example/forumgo/internal/store"
)

func TestThreadListLockedQueryExcludesLockedThreads(t *testing.T) {
	boardStore, _ := store.NewMemoryBoardStore("", nil, 30*time.Second)
	threadStore, _ := store.NewMemoryThreadStore("", nil, 30*time.Second)
	t.Cleanup(func() {
		boardStore.Close()
		threadStore.Close()
	})
	boardSvc := service.NewBoardService(boardStore)
	threadSvc := service.NewThreadService(threadStore, boardStore)
	boardSvc.Create(context.Background(), &model.Board{ID: "b1", Name: "Board", Slug: "board"})
	open, _ := threadSvc.Create(context.Background(), &model.Thread{BoardID: "b1", AuthorID: "u1", Title: "Open thread", Body: "body"})
	locked, _ := threadSvc.Create(context.Background(), &model.Thread{BoardID: "b1", AuthorID: "u2", Title: "Locked thread", Body: "body"})
	threadSvc.Update(context.Background(), locked.ID, &service.UpdateThreadRequest{IsLocked: ptrBoolHTTP(true)})

	handler := NewThreadHandler(threadSvc, 1<<20)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/threads?locked=false", nil)
	rec := httptest.NewRecorder()
	handler.List(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got []model.Thread
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 1 || got[0].ID != open.ID {
		t.Fatalf("threads = %#v, want only unlocked thread %s", got, open.ID)
	}
}

func ptrBoolHTTP(v bool) *bool { return &v }
