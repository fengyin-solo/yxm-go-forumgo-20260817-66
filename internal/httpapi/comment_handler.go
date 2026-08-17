package httpapi

import (
	"net/http"

	"github.com/example/forumgo/internal/model"
	"github.com/example/forumgo/internal/service"
	"github.com/example/forumgo/internal/validator"
)

// CommentHandler handles comment HTTP requests.
type CommentHandler struct {
	svc      *service.CommentService
	voteSvc  *service.VoteService
	reportSvc *service.ReportService
	maxBody  int64
}

// NewCommentHandler creates a new CommentHandler.
func NewCommentHandler(svc *service.CommentService, voteSvc *service.VoteService, reportSvc *service.ReportService, maxBody int64) *CommentHandler {
	return &CommentHandler{svc: svc, voteSvc: voteSvc, reportSvc: reportSvc, maxBody: maxBody}
}

// Create handles POST /api/v1/comments.
func (h *CommentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req validator.CreateCommentRequest
	if err := decodeJSON(r, &req, h.maxBody); err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	if ve := req.Validate(); len(ve) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "validation failed", "details": ve})
		return
	}
	c := &model.Comment{
		ThreadID: req.ThreadID,
		ParentID: req.ParentID,
		AuthorID: req.AuthorID,
		Body:     req.Body,
	}
	created, err := h.svc.Create(r.Context(), c)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// Get handles GET /api/v1/comments/{id}.
func (h *CommentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := pathSegment(r.URL.Path, 4)
	c, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	if c.IsDeleted {
		writeError(w, model.ErrNotFound)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// Update handles PUT /api/v1/comments/{id}.
func (h *CommentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := pathSegment(r.URL.Path, 4)
	var req service.UpdateCommentRequest
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

// Delete handles DELETE /api/v1/comments/{id}.
func (h *CommentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := pathSegment(r.URL.Path, 4)
	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListByThread handles GET /api/v1/threads/{threadID}/comments.
func (h *CommentHandler) ListByThread(w http.ResponseWriter, r *http.Request) {
	threadID := pathSegment(r.URL.Path, 4)
	q := r.URL.Query()
	f := model.CommentFilter{ThreadID: threadID}
	if v := q.Get("top_level"); v == "true" {
		b := true
		f.ParentID = &b
	} else if v == "false" {
		b := false
		f.ParentID = &b
	}
	if v := q.Get("sort"); v != "" {
		f.SortBy = v
	}
	list, err := h.svc.List(r.Context(), f)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// Vote handles POST /api/v1/threads/{id}/vote and POST /api/v1/comments/{id}/vote.
func (h *CommentHandler) Vote(w http.ResponseWriter, r *http.Request) {
	pathParts := splitPath(r.URL.Path)
	var targetType, targetID string
	if len(pathParts) >= 4 && pathParts[2] == "threads" {
		targetType = "thread"
		targetID = pathParts[3]
	} else if len(pathParts) >= 4 && pathParts[2] == "comments" {
		targetType = "comment"
		targetID = pathParts[3]
	}
	var req struct {
		UserID string `json:"user_id"`
		Value  int    `json:"value"`
	}
	if err := decodeJSON(r, &req, h.maxBody); err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	v := &model.Vote{
		TargetType: targetType,
		TargetID:   targetID,
		UserID:     req.UserID,
		Value:      req.Value,
	}
	if err := h.voteSvc.Vote(r.Context(), v); err != nil {
		writeError(w, err)
		return
	}
	up, down, _ := h.voteSvc.GetVotes(r.Context(), targetType, targetID)
	writeJSON(w, http.StatusOK, map[string]int{"upvotes": up, "downvotes": down})
}

// Report handles POST /api/v1/reports.
func (h *CommentHandler) Report(w http.ResponseWriter, r *http.Request) {
	var req validator.CreateReportRequest
	if err := decodeJSON(r, &req, h.maxBody); err != nil {
		writeError(w, model.ErrInvalidInput)
		return
	}
	if ve := req.Validate(); len(ve) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "validation failed", "details": ve})
		return
	}
	rep := &model.Report{
		ReporterID: req.ReporterID,
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
		Reason:     req.Reason,
	}
	created, err := h.reportSvc.Create(r.Context(), rep)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}
