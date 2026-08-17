package httpapi

import (
	"net/http"

	"github.com/example/forumgo/internal/auth"
)

// Router holds all HTTP handlers.
type Router struct {
	boards   *BoardHandler
	threads  *ThreadHandler
	comments *CommentHandler
}

// NewRouter creates a new Router.
func NewRouter(boards *BoardHandler, threads *ThreadHandler, comments *CommentHandler) *Router {
	return &Router{boards: boards, threads: threads, comments: comments}
}

// Handler returns the top-level HTTP handler with all routes registered.
func (rt *Router) Handler(authenticator *auth.Authenticator) http.Handler {
	mux := http.NewServeMux()

	// Boards
	mux.HandleFunc("GET /api/v1/boards", rt.boards.List)
	mux.HandleFunc("POST /api/v1/boards", rt.boards.Create)
	mux.HandleFunc("GET /api/v1/boards/by-slug/{slug}", rt.boards.GetBySlug)
	mux.HandleFunc("GET /api/v1/boards/{id}", rt.boards.Get)
	mux.HandleFunc("PUT /api/v1/boards/{id}", rt.boards.Update)
	mux.HandleFunc("DELETE /api/v1/boards/{id}", rt.boards.Delete)
	mux.HandleFunc("GET /api/v1/boards/{boardID}/threads", rt.threads.ListByBoard)

	// Threads
	mux.HandleFunc("GET /api/v1/threads", rt.threads.List)
	mux.HandleFunc("POST /api/v1/threads", rt.threads.Create)
	mux.HandleFunc("GET /api/v1/threads/{id}", rt.threads.Get)
	mux.HandleFunc("PUT /api/v1/threads/{id}", rt.threads.Update)
	mux.HandleFunc("DELETE /api/v1/threads/{id}", rt.threads.Delete)

	// Comments
	mux.HandleFunc("GET /api/v1/threads/{threadID}/comments", rt.comments.ListByThread)
	mux.HandleFunc("POST /api/v1/comments", rt.comments.Create)
	mux.HandleFunc("GET /api/v1/comments/{id}", rt.comments.Get)
	mux.HandleFunc("PUT /api/v1/comments/{id}", rt.comments.Update)
	mux.HandleFunc("DELETE /api/v1/comments/{id}", rt.comments.Delete)

	// Votes (thread or comment)
	mux.HandleFunc("POST /api/v1/threads/{id}/vote", rt.comments.Vote)
	mux.HandleFunc("POST /api/v1/comments/{id}/vote", rt.comments.Vote)

	// Reports
	mux.HandleFunc("POST /api/v1/reports", rt.comments.Report)

	return mux
}
