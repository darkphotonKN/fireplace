package checklistitem

import (
	"context"
	"fmt"
	"testing"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/plan"
	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FS-PGG2N R2, R3, R5: the handler forwards summary as set-or-leave, returns it
// when the column holds one, and maps a refusal to InvalidArgument.

// summaryRecorder keeps the UpdateItemInput the handler built and answers with
// an item carrying the summary it was told to return.
type summaryRecorder struct {
	Service

	got   UpdateItemInput
	reply *string
	err   error
}

func (r *summaryRecorder) Update(ctx context.Context, in UpdateItemInput) (*Item, error) {
	r.got = in
	if r.err != nil {
		return nil, r.err
	}
	return &Item{ID: in.ID, PlanID: in.PlanID, Type: TypeNote, Summary: r.reply}, nil
}

func TestUpdateItem_Summary_ForwardsSetOrLeave(t *testing.T) {
	id, planID := uuid.New(), uuid.New()
	cases := []struct {
		name    string
		in      *string
		wantSet bool
	}{
		{"unset leaves it alone", nil, false},
		{"a string sets it", strPtr("  gist  "), true},
		{"an empty string is still a set", strPtr(""), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &summaryRecorder{}
			_, err := NewHandler(rec).UpdateItem(context.Background(), &pb.UpdateItemRequest{
				Id: id.String(), PlanId: planID.String(), Summary: tc.in,
			})
			if err != nil {
				t.Fatalf("UpdateItem: %v", err)
			}
			if rec.got.SetSummary != tc.wantSet {
				t.Errorf("SetSummary = %v, want %v", rec.got.SetSummary, tc.wantSet)
			}
			if tc.wantSet && (rec.got.Summary == nil || *rec.got.Summary != *tc.in) {
				t.Errorf("Summary = %v, want %q passed through untrimmed for the service", rec.got.Summary, *tc.in)
			}
		})
	}
}

func TestUpdateItem_Summary_IsReturnedWhenSetAndAbsentWhenNull(t *testing.T) {
	req := &pb.UpdateItemRequest{Id: uuid.NewString(), PlanId: uuid.NewString()}

	got, err := NewHandler(&summaryRecorder{reply: strPtr("gist")}).UpdateItem(context.Background(), req)
	if err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}
	if got.Summary == nil || *got.Summary != "gist" {
		t.Errorf("summary = %v, want \"gist\"", got.Summary)
	}

	got, err = NewHandler(&summaryRecorder{}).UpdateItem(context.Background(), req)
	if err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}
	if got.Summary != nil {
		t.Errorf("summary = %q, want unset", *got.Summary)
	}
}

func TestUpdateItem_Summary_RefusalIsInvalidArgument(t *testing.T) {
	rec := &summaryRecorder{err: fmt.Errorf("%w: only a note can carry a summary", commonconstants.ErrInvalidInput)}
	_, err := NewHandler(rec).UpdateItem(context.Background(), &pb.UpdateItemRequest{
		Id: uuid.NewString(), PlanId: uuid.NewString(), Summary: strPtr("gist"),
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", status.Code(err))
	}
}
