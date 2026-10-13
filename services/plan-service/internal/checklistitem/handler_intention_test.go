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

// intentionRecorder keeps the UpdateItemInput the handler built and answers
// with an item carrying the intention it was told to return.
type intentionRecorder struct {
	Service

	got   UpdateItemInput
	reply *string
	err   error
}

func (r *intentionRecorder) Update(ctx context.Context, in UpdateItemInput) (*Item, error) {
	r.got = in
	if r.err != nil {
		return nil, r.err
	}
	return &Item{ID: in.ID, PlanID: in.PlanID, Intention: r.reply}, nil
}

func TestUpdateItem_Intention_ForwardsSetOrLeave(t *testing.T) {
	id, planID := uuid.New(), uuid.New()
	value := "  ship it  "
	cases := []struct {
		name    string
		in      *string
		wantSet bool
	}{
		{"unset leaves it alone", nil, false},
		{"a string sets it", &value, true},
		{"an empty string is still a set", strPtr(""), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &intentionRecorder{}
			_, err := NewHandler(rec).UpdateItem(context.Background(), &pb.UpdateItemRequest{
				Id: id.String(), PlanId: planID.String(), Intention: tc.in,
			})
			if err != nil {
				t.Fatalf("UpdateItem: %v", err)
			}
			if rec.got.SetIntention != tc.wantSet {
				t.Errorf("SetIntention = %v, want %v", rec.got.SetIntention, tc.wantSet)
			}
			if tc.wantSet && (rec.got.Intention == nil || *rec.got.Intention != *tc.in) {
				t.Errorf("Intention = %v, want %q passed through untrimmed for the service", rec.got.Intention, *tc.in)
			}
		})
	}
}

func TestUpdateItem_Intention_IsReturnedWhenSetAndAbsentWhenNull(t *testing.T) {
	req := &pb.UpdateItemRequest{Id: uuid.NewString(), PlanId: uuid.NewString()}

	got, err := NewHandler(&intentionRecorder{reply: strPtr("why")}).UpdateItem(context.Background(), req)
	if err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}
	if got.Intention == nil || *got.Intention != "why" {
		t.Errorf("intention = %v, want \"why\"", got.Intention)
	}

	got, err = NewHandler(&intentionRecorder{}).UpdateItem(context.Background(), req)
	if err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}
	if got.Intention != nil {
		t.Errorf("intention = %q, want unset", *got.Intention)
	}
}

func TestUpdateItem_Intention_TooLongIsInvalidArgument(t *testing.T) {
	rec := &intentionRecorder{err: fmt.Errorf("%w: intention too long", commonconstants.ErrInvalidInput)}
	_, err := NewHandler(rec).UpdateItem(context.Background(), &pb.UpdateItemRequest{
		Id: uuid.NewString(), PlanId: uuid.NewString(), Intention: strPtr("x"),
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", status.Code(err))
	}
}
