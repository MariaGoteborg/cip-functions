package combinedsewageoverflow

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/diwise/cip-functions/internal/pkg/application/things"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/google/uuid"
)

var CombinedSewageOverflowFactory = func(id, tenant string) *CombinedSewageOverflow {
	return &CombinedSewageOverflow{
		ID:     id,
		Type:   "CombinedSewageOverflow",
		Tenant: tenant,
		DateObserved: time.Now().UTC(),		
	}
}

type CombinedSewageOverflow struct {
	ID                     string        `json:"id"`
	Type                   string        `json:"type"`
	CumulativeTime         time.Duration `json:"cumulativeTime"`
	DateObserved           time.Time     `json:"dateObserved"`
	Overflows              []Overflow    `json:"overflow"`
	State                  bool          `json:"state"`
	Tenant                 string        `json:"tenant"`
	CombinedSewageOverflow *things.Thing `json:"combinedsewageoverflow,omitempty"`
}

type Overflow struct {
	ID        string        `json:"id"`
	State     bool          `json:"state"`
	StartTime time.Time     `json:"startTime"`
	StopTime  *time.Time    `json:"stopTime"`
	Duration  time.Duration `json:"duration"`
}

type functionUpdated struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	SubType   string    `json:"subtype"`
	Stopwatch stopwatch `json:"stopwatch"`
}

type stopwatch struct {
	StartTime time.Time      `json:"startTime"`
	StopTime  *time.Time     `json:"stopTime,omitempty"`
	Duration  *time.Duration `json:"duration,omitempty"`

	State bool  `json:"state"`
	Count int32 `json:"count"`

	CumulativeTime time.Duration `json:"cumulativeTime"`
}

func (cso CombinedSewageOverflow) TopicName() string {
	return "cip-function.updated"
}

func (cso CombinedSewageOverflow) ContentType() string {
	return "application/vnd.diwise.combinedsewageoverflow+json"
}

func (cso CombinedSewageOverflow) Body() []byte {
	b, _ := json.Marshal(cso)
	return b
}

func getStopwatch(itm messaging.IncomingTopicMessage) (*stopwatch, error) {
	stopwatch := functionUpdated{}

	err := json.Unmarshal(itm.Body(), &stopwatch)
	if err != nil {
		return nil, err
	}

	return &stopwatch.Stopwatch, nil
}

func (cso *CombinedSewageOverflow) Handle(ctx context.Context, itm messaging.IncomingTopicMessage, tc things.Client) (bool, error) {
	changed := false

	sw, err := getStopwatch(itm)
	if err != nil {
		return false, err
	}

	if sw.StartTime.IsZero() {
		return false, fmt.Errorf("start time is zero")
	}

	if cso.CombinedSewageOverflow == nil {
		if t, err := tc.FindByID(ctx, cso.ID); err == nil {
			cso.CombinedSewageOverflow = &t
		}
	}

	i, changed := getIndexForOverflow(cso, sw.StartTime)

	overflow := &cso.Overflows[i]

	if overflow.StopTime != nil {
		return changed, fmt.Errorf("current overflow already ended")
	}

	if overflow.State != sw.State {
		if sw.State {
			if sw.Duration != nil {
				overflow.Duration = *sw.Duration
			}

			if sw.Duration == nil {
				overflow.Duration = time.Now().UTC().Sub(overflow.StartTime.UTC())
			}

			cso.DateObserved = overflow.StartTime.UTC()
		}

		if !sw.State {
			overflow.StopTime = sw.StopTime

			if sw.Duration != nil {
				overflow.Duration = *sw.Duration
			}

			if sw.Duration == nil {
				overflow.Duration = overflow.StopTime.UTC().Sub(overflow.StartTime.UTC())
			}

			cso.DateObserved = overflow.StopTime.UTC()
		}

		overflow.State = sw.State
		changed = true
	}

	slices.SortFunc(cso.Overflows, func(a, b Overflow) int {
		return a.StartTime.Compare(b.StartTime)
	})

	cumulativeTime := 0
	for _, o := range cso.Overflows {
		cumulativeTime += int(o.Duration)
	}

	if cso.CumulativeTime != time.Duration(cumulativeTime) {
		cso.CumulativeTime = time.Duration(cumulativeTime)
		changed = true
	}

	if cso.State != cso.Overflows[len(cso.Overflows)-1].State {
		cso.State = cso.Overflows[len(cso.Overflows)-1].State
		changed = true
	}

	return changed, nil
}

func getIndexForOverflow(cso *CombinedSewageOverflow, t time.Time) (int, bool) {
	overflowID := deterministicUUID(t)

	idx := slices.IndexFunc(cso.Overflows, func(o Overflow) bool {
		return o.ID == overflowID
	})

	if idx >= 0 {
		return idx, false
	}

	overflow := Overflow{ID: overflowID, StartTime: t.UTC(), State: false, Duration: 0}
	cso.Overflows = append(cso.Overflows, overflow)

	idx = slices.IndexFunc(cso.Overflows, func(o Overflow) bool {
		return o.ID == overflowID
	})

	return idx, true
}

func deterministicUUID(t time.Time) string {
	str := fmt.Sprintf("%d", t.UnixNano())

	md5hash := md5.New()
	md5hash.Write([]byte(str))
	md5string := hex.EncodeToString(md5hash.Sum(nil))

	unique, err := uuid.FromBytes([]byte(md5string[0:16]))
	if err != nil {
		return uuid.New().String()
	}

	return unique.String()
}
