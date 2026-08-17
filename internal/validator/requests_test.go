package validator

import (
	"testing"

	"github.com/example/forumgo/internal/model"
)

func TestCreateBoardRequest(t *testing.T) {
	r := CreateBoardRequest{Name: "  General  "}
	ve := r.Validate()
	if len(ve) > 0 {
		t.Errorf("should not validate with name: %v", ve)
	}
	r2 := CreateBoardRequest{Name: "  "}
	ve2 := r2.Validate()
	if len(ve2) == 0 {
		t.Error("empty name should fail validation")
	}
}

func TestCreateThreadRequest(t *testing.T) {
	r := CreateThreadRequest{BoardID: "b1", Title: "Hello"}
	ve := r.Validate()
	if len(ve) > 0 {
		t.Errorf("should not fail: %v", ve)
	}
	r2 := CreateThreadRequest{}
	ve2 := r2.Validate()
	if len(ve2) < 2 {
		t.Errorf("should fail for missing fields: %v", ve2)
	}
}

func TestCreateCommentRequest(t *testing.T) {
	r := CreateCommentRequest{ThreadID: "t1", Body: "  Nice!  "}
	ve := r.Validate()
	if len(ve) > 0 {
		t.Errorf("should not fail: %v", ve)
	}
	r2 := CreateCommentRequest{Body: ""}
	ve2 := r2.Validate()
	if len(ve2) == 0 {
		t.Error("empty body should fail")
	}
}

func TestCreateVoteRequest(t *testing.T) {
	r := CreateVoteRequest{TargetType: "thread", TargetID: "t1", UserID: "u1", Value: 1}
	ve := r.Validate()
	if len(ve) > 0 {
		t.Errorf("should not fail: %v", ve)
	}
	r2 := CreateVoteRequest{TargetType: "invalid", Value: 2}
	ve2 := r2.Validate()
	if len(ve2) == 0 {
		t.Error("invalid fields should fail")
	}
}

func TestCreateReportRequest(t *testing.T) {
	r := CreateReportRequest{TargetType: "comment", TargetID: "c1", Reason: "spam"}
	ve := r.Validate()
	if len(ve) > 0 {
		t.Errorf("should not fail: %v", ve)
	}
	r2 := CreateReportRequest{}
	ve2 := r2.Validate()
	if len(ve2) < 3 {
		t.Errorf("should fail for missing fields: %v", ve2)
	}
}

func TestValidationErrors(t *testing.T) {
	ve := model.ValidationErrors{
		{Field: "name", Message: "required"},
		{Field: "age", Message: "must be positive"},
	}
	if len(ve) != 2 {
		t.Errorf("len = %d", len(ve))
	}
	errStr := ve.Error()
	if errStr == "" {
		t.Error("Error() should return non-empty string")
	}
}
