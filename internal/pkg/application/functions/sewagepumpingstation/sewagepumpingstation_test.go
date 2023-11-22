package sewagepumpingstation

import (
	"context"
	"testing"
	"time"

	"github.com/diwise/cip-functions/internal/pkg/infrastructure/database"
	"github.com/diwise/cip-functions/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/matryer/is"
)

func TestSewagePumpingStationHandleCreatesNewIfIDDoesNotExist(t *testing.T) {
	is, dbMock, msgCtxMock := testSetup(t)

	msg := events.FunctionUpdated{
		ID:   "fnID:003",
		Type: "Stopwatch",
		Stopwatch: struct {
			Count          int32          "json:\"count\""
			CumulativeTime time.Duration  "json:\"cumulativeTime\""
			Duration       *time.Duration "json:\"duration,omitempty\""
			StartTime      time.Time      "json:\"startTime\""
			State          bool           "json:\"state\""
			StopTime       *time.Time     "json:\"stopTime,omitempty\""
		}{
			State:     false,
			StartTime: time.Now(),
		},
		Timestamp: time.Now(),
	}

	sp := New()
	err := sp.Handle(context.Background(), &msg, dbMock, msgCtxMock)

	is.NoErr(err)
	is.True(len(dbMock.CreateCalls()) == 1)
}

func TestSewagePumpingStationHandleChecksIfStateUpdatedOnExisting(t *testing.T) {
	is, dbMock, msgCtxMock := testSetup(t)

	msg := events.FunctionUpdated{
		ID:   "fnID:004",
		Type: "Stopwatch",
		Stopwatch: struct {
			Count          int32          "json:\"count\""
			CumulativeTime time.Duration  "json:\"cumulativeTime\""
			Duration       *time.Duration "json:\"duration,omitempty\""
			StartTime      time.Time      "json:\"startTime\""
			State          bool           "json:\"state\""
			StopTime       *time.Time     "json:\"stopTime,omitempty\""
		}{
			State:     false,
			StartTime: time.Now(),
		},
		Timestamp: time.Now(),
	}

	//create new entry first time around
	sp := New()
	err := sp.Handle(context.Background(), &msg, dbMock, msgCtxMock)
	is.NoErr(err)

	//update value on state
	msg.Stopwatch.State = true

	//call New and Handle again with new value
	sp2 := New()
	err = sp2.Handle(context.Background(), &msg, dbMock, msgCtxMock)

	is.NoErr(err)
	is.True(len(dbMock.CreateCalls()) == 1)
	is.True(len(dbMock.UpdateCalls()) == 2)
}

func testSetup(t *testing.T) (*is.I, *database.StorageMock, *messaging.MsgContextMock) {
	is := is.New(t)

	dbMock := &database.StorageMock{
		ExistsFunc: func(ctx context.Context, id string) bool {
			if id == "SewagePumpingStationObserved:fnID:004" {
				return true
			} else {
				return false
			}
		},
		CreateFunc: func(ctx context.Context, id string, value any) error {
			return nil
		},
		UpdateFunc: func(ctx context.Context, id string, value any) error {
			return nil
		},
		SelectFunc: func(ctx context.Context, id string) (any, error) {
			return SewagePumpingStationObserved{
				ID:          "SewagePumpingStationObserved:fnID:004",
				ActiveAlert: "",
				State:       false,
			}, nil
		},
	}

	msgCtxMock := &messaging.MsgContextMock{}

	return is, dbMock, msgCtxMock
}
