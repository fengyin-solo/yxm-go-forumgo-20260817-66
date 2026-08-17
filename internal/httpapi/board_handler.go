package httpapi

import (
	"net/http"
	"strings"

	"github.com/example/forumgo/internal/model"
	"github.com/example/forumgo/internal/service"
	"github.com/example/forumgo/internal/validator"
)

// BoardHandler handles board HTTP requests.
type BoardHandler struct {
	svc     *service.BoardService
	maxBody int64
}

// NewBoardHandler creates a new BoardHandler.
func NewBoardHandler(svc *service.BoardService, maxBody int64) *BoardHandler {
	return &BoardHandler{svc: svc, maxBody: maxBody}
}

// List handles GET /api/v1/boards.
func (h *BoardHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := model.BoardFilter{}
	if v := q.Get("q"); v != "" {
		f.Query = v
	}
	list, err := h.svc.List(r.Context(), f)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// Create handles POST /api/v1/boards.
func (h *BoardHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req validator.CreateBoardRequest
	if err := decodeJSON(r, &req, h.maxBody); err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	if ve := req.Validate(); len(ve) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "validation failed", "details": ve})
		return
	}
	b := &model.Board{
		Name:         req.Name,
		Slug:         req.Slug,
		Description:  req.Description,
		DisplayOrder: req.DisplayOrder,
	}
	created, err := h.svc.Create(r.Context(), b)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// Get handles GET /api/v1/boards/{id}.
func (h *BoardHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := pathSegment(r.URL.Path, 4)
	b, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// Update handles PUT /api/v1/boards/{id}.
func (h *BoardHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := pathSegment(r.URL.Path, 4)
	var req service.UpdateBoardRequest
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

// Delete handles DELETE /api/v1/boards/{id}.
func (h *BoardHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := pathSegment(r.URL.Path, 4)
	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetBySlug handles GET /api/v1/boards/by-slug/{slug}.
func (h *BoardHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := pathSegment(r.URL.Path, 5)
	b, err := h.svc.GetBySlug(r.Context(), slug)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// parseBoardFilter extracts a BoardFilter from query params.
func parseBoardFilter(q map[string][]string) model.BoardFilter {
	f := model.BoardFilter{}
	if v := q["q"]; len(v) > 0 {
		f.Query = strings.Join(v, " ")
	}
	return f
}
