package validator

import "github.com/example/forumgo/internal/model"

// CreateBoardRequest is the payload for creating a board.
type CreateBoardRequest struct {
	Name         string `json:"name"`
	Slug         string `json:"slug,omitempty"`
	Description  string `json:"description,omitempty"`
	DisplayOrder int    `json:"display_order"`
}

// Validate checks the request and returns any errors.
func (r CreateBoardRequest) Validate() (ve model.ValidationErrors) {
	if r.Name = trimSpace(r.Name); r.Name == "" {
		ve = append(ve, model.ValidationError{Field: "name", Message: "required"})
	}
	return ve
}

// CreateThreadRequest is the payload for creating a thread.
type CreateThreadRequest struct {
	BoardID  string `json:"board_id"`
	AuthorID string `json:"author_id"`
	Title    string `json:"title"`
	Body     string `json:"body"`
}

// Validate checks the request and returns any errors.
func (r CreateThreadRequest) Validate() (ve model.ValidationErrors) {
	if r.Title = trimSpace(r.Title); r.Title == "" {
		ve = append(ve, model.ValidationError{Field: "title", Message: "required"})
	}
	if r.BoardID = trimSpace(r.BoardID); r.BoardID == "" {
		ve = append(ve, model.ValidationError{Field: "board_id", Message: "required"})
	}
	return ve
}

// CreateCommentRequest is the payload for creating a comment.
type CreateCommentRequest struct {
	ThreadID string  `json:"thread_id"`
	ParentID *string `json:"parent_id,omitempty"`
	AuthorID string  `json:"author_id"`
	Body     string  `json:"body"`
}

// Validate checks the request and returns any errors.
func (r CreateCommentRequest) Validate() (ve model.ValidationErrors) {
	if r.Body = trimSpace(r.Body); r.Body == "" {
		ve = append(ve, model.ValidationError{Field: "body", Message: "required"})
	}
	if r.ThreadID = trimSpace(r.ThreadID); r.ThreadID == "" {
		ve = append(ve, model.ValidationError{Field: "thread_id", Message: "required"})
	}
	return ve
}

// CreateVoteRequest is the payload for creating a vote.
type CreateVoteRequest struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	UserID     string `json:"user_id"`
	Value      int    `json:"value"`
}

// Validate checks the request and returns any errors.
func (r CreateVoteRequest) Validate() (ve model.ValidationErrors) {
	if r.TargetType = trimSpace(r.TargetType); r.TargetType != "thread" && r.TargetType != "comment" {
		ve = append(ve, model.ValidationError{Field: "target_type", Message: "must be thread or comment"})
	}
	if r.TargetID = trimSpace(r.TargetID); r.TargetID == "" {
		ve = append(ve, model.ValidationError{Field: "target_id", Message: "required"})
	}
	if r.UserID = trimSpace(r.UserID); r.UserID == "" {
		ve = append(ve, model.ValidationError{Field: "user_id", Message: "required"})
	}
	if r.Value != 1 && r.Value != -1 {
		ve = append(ve, model.ValidationError{Field: "value", Message: "must be 1 or -1"})
	}
	return ve
}

// CreateReportRequest is the payload for creating a report.
type CreateReportRequest struct {
	ReporterID string `json:"reporter_id"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	Reason     string `json:"reason"`
}

// Validate checks the request and returns any errors.
func (r CreateReportRequest) Validate() (ve model.ValidationErrors) {
	if r.Reason = trimSpace(r.Reason); r.Reason == "" {
		ve = append(ve, model.ValidationError{Field: "reason", Message: "required"})
	}
	if r.TargetType = trimSpace(r.TargetType); r.TargetType != "thread" && r.TargetType != "comment" {
		ve = append(ve, model.ValidationError{Field: "target_type", Message: "must be thread or comment"})
	}
	if r.TargetID = trimSpace(r.TargetID); r.TargetID == "" {
		ve = append(ve, model.ValidationError{Field: "target_id", Message: "required"})
	}
	return ve
}

func trimSpace(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\t' && s[i] != '\n' && s[i] != '\r' {
			b = append(b, s[i])
		}
	}
	return string(b)
}
