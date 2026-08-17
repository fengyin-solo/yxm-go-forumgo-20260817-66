package httpapi

import (
	"net/http"
	"strings"

	"github.com/example/forumgo/internal/model"
	"github.com/example/forumgo/internal/service"
	"github.com/example/forumgo/internal/validator"
)

// ThreadHandler handles thread HTTP requests.
type ThreadHandler struct {
	svc     *service.ThreadService
	maxBody int64
}

// NewThreadHandler creates a new ThreadHandler.
func NewThreadHandler(svc *service.ThreadService, maxBody int64) *ThreadHandler {
	return &ThreadHandler{svc: svc, maxBody: maxBody}
}

// List handles GET /api/v1/threads.
func (h *ThreadHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := model.ThreadFilter{}
	if v := q.Get("board_id"); v != "" {
		f.BoardID = v
	}
	if v := q.Get("author_id"); v != "" {
		f.AuthorID = v
	}
	if v := q.Get("q"); v != "" {
		f.Query = v
	}
	if v := q.Get("pinned"); v == "true" {
		b := true
		f.IsPinned = &b
	} else if v == "false" {
		b := false
		f.IsPinned = &b
	}
	if v := q.Get("locked"); v == "true" {
		b := true
		f.IsLocked = &b
	} else if v == "false" {
		b := false
		f.IsLocked = &b
	}
	list, err := h.svc.List(r.Context(), f)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// Create handles POST /api/v1/threads.
func (h *ThreadHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req validator.CreateThreadRequest
	if err := decodeJSON(r, &req, h.maxBody); err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	if ve := req.Validate(); len(ve) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "validation failed", "details": ve})
		return
	}
	t := &model.Thread{
		BoardID:  req.BoardID,
		AuthorID: req.AuthorID,
		Title:    req.Title,
		Body:     req.Body,
	}
	created, err := h.svc.Create(r.Context(), t)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// Get handles GET /api/v1/threads/{id}.
func (h *ThreadHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := pathSegment(r.URL.Path, 4)
	t, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// Update handles PUT /api/v1/threads/{id}.
func (h *ThreadHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := pathSegment(r.URL.Path, 4)
	var req service.UpdateThreadRequest
	if err := decodeJSON(r, &req, h.maxBody); err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	updated, err := h.svc.Update(r.Context(), id, &req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// Delete handles DELETE /api/v1/threads/{id}.
func (h *ThreadHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := pathSegment(r.URL.Path, 4)
	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListByBoard handles GET /api/v1/boards/{boardID}/threads.
func (h *ThreadHandler) ListByBoard(w http.ResponseWriter, r *http.Request) {
	boardID := pathSegment(r.URL.Path, 4)
	q := r.URL.Query()
	f := model.ThreadFilter{BoardID: boardID}
	if v := q.Get("q"); v != "" {
		f.Query = v
	}
	if v := q.Get("pinned"); v == "true" {
		b := true
		f.IsPinned = &b
	} else if v == "false" {
		b := false
		f.IsPinned = &b
	}
	if v := q.Get("locked"); v == "true" {
		b := true
		f.IsLocked = &b
	} else if v == "false" {
		b := false
		f.IsLocked = &b
	}
	list, err := h.svc.List(r.Context(), f)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// parseThreadFilter extracts a ThreadFilter from query params.
func parseThreadFilter(q map[string][]string) model.ThreadFilter {
	f := model.ThreadFilter{}
	if v := q["board_id"]; len(v) > 0 {
		f.BoardID = v[0]
	}
	if v := q["author_id"]; len(v) > 0 {
		f.AuthorID = v[0]
	}
	if v := q["q"]; len(v) > 0 {
		f.Query = strings.Join(v, " ")
	}
	return f
}
