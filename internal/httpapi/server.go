package httpapi

import (
	"net/http"
	"time"

	"github.com/example/forumgo/internal/auth"
	"github.com/example/forumgo/internal/config"
	"github.com/example/forumgo/internal/logger"
	"github.com/example/forumgo/internal/middleware"
	"github.com/example/forumgo/internal/service"
	"github.com/example/forumgo/internal/store"
)

// Server wires together all components and runs the HTTP server.
type Server struct {
	httpServer *http.Server
	boardSvc   *service.BoardService
	threadSvc  *service.ThreadService
	commentSvc *service.CommentService
	voteSvc    *service.VoteService
	reportSvc  *service.ReportService
	auth       *auth.Authenticator
	log        *logger.Logger
}

// NewServer creates a new Server.
func NewServer(cfg *config.Config) *Server {
	log := logger.Default()

	// Create stores
	boardStore, _ := store.NewMemoryBoardStore(cfg.DataDir+"/boards.json", log, 30*time.Second)
	threadStore, _ := store.NewMemoryThreadStore(cfg.DataDir+"/threads.json", log, 30*time.Second)
	commentStore, _ := store.NewMemoryCommentStore(cfg.DataDir+"/comments.json", log, 30*time.Second)
	voteStore, _ := store.NewMemoryVoteStore(cfg.DataDir+"/votes.json", log, 30*time.Second)
	reportStore, _ := store.NewMemoryReportStore(cfg.DataDir+"/reports.json", log, 30*time.Second)

	// Create services
	boardSvc := service.NewBoardService(boardStore)
	threadSvc := service.NewThreadService(threadStore, boardStore)
	commentSvc := service.NewCommentService(commentStore, threadStore)
	voteSvc := service.NewVoteService(voteStore, threadStore, commentStore)
	reportSvc := service.NewReportService(reportStore)

	// Create handlers
	boardHandler := NewBoardHandler(boardSvc, cfg.MaxBody)
	threadHandler := NewThreadHandler(threadSvc, cfg.MaxBody)
	commentHandler := NewCommentHandler(commentSvc, voteSvc, reportSvc, cfg.MaxBody)

	// Create router
	router := NewRouter(boardHandler, threadHandler, commentHandler)
	mux := router.Handler(nil)

	// Wrap with middleware
	authenticator := auth.NewAuthenticator()
	stack := middleware.RequestID(mux)
	stack = middleware.Logging(stack, log)
	stack = middleware.Recovery(stack, log)
	if cfg.AuthToken != "" {
		stack = middleware.RequireAuth(stack, authenticator, log)
	}
	stack = middleware.Timeout(stack, cfg.Timeout)
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimit, cfg.RateWindow)
	stack = rateLimiter.Limit(stack)

	return &Server{
		httpServer: &http.Server{
			Addr:         cfg.Addr,
			Handler:      stack,
			ReadTimeout:  cfg.Timeout,
			WriteTimeout: cfg.Timeout,
		},
		boardSvc:   boardSvc,
		threadSvc:  threadSvc,
		commentSvc: commentSvc,
		voteSvc:    voteSvc,
		reportSvc:  reportSvc,
		auth:       authenticator,
		log:        log,
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.log.Info("starting server", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Close gracefully shuts down the server.
func (s *Server) Close() error {
	return s.httpServer.Close()
}

// Authenticator returns the server's authenticator.
func (s *Server) Authenticator() *auth.Authenticator {
	return s.auth
}
