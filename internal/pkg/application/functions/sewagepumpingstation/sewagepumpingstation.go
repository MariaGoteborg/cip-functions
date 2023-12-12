package sewagepumpingstation

import (
	"context"
	"fmt"
	"time"

	"github.com/diwise/cip-functions/internal/pkg/application/functions/options"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/database"
	"github.com/diwise/cip-functions/pkg/messaging/events"
	"github.com/diwise/cip-functions/pkg/messaging/topics"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

const FunctionName string = "sewagepumpingstation"

type SewagePumpingStation struct {
	ID    string `json:"id"`
	State bool   `json:"state"`

	StartTime *time.Time `json:"startTime,omitempty"`
	EndTime   *time.Time `json:"endTime,omitempty"`

	ObservedAt *time.Time `json:"observedAt"`
}

func New() SewagePumpingStation {
	return SewagePumpingStation{}
}

func (sp SewagePumpingStation) Body() []byte {
	return []byte{}
}

func (sp SewagePumpingStation) TopicName() string {
	return topics.CipFunctionsUpdated
}

func (sp SewagePumpingStation) ContentType() string {
	return "application/vnd+diwise.sewagepumpingstation+json"
}

func (sp *SewagePumpingStation) Handle(ctx context.Context, msg *events.FunctionUpdated, storage database.Storage, msgCtx messaging.MsgContext, opts ...options.Option) error {

	log := logging.GetFromContext(ctx)

	if msg.Type != "stopwatch" {
		log.Info("invalid function type", "id", msg.ID, "type", msg.Type, "sub_type", msg.SubType)
		return nil
	}

	id := fmt.Sprintf("sewagepumpingstation:%s", msg.ID)

	exists := storage.Exists(ctx, id)
	if !exists {
		spo := SewagePumpingStation{
			ID:         id,
			State:      msg.Stopwatch.State,
			ObservedAt: &msg.Timestamp,
		}

		if msg.Stopwatch.State {
			if msg.Stopwatch.StartTime.IsZero() {
				log.Error("invalid stopwatch message", msg.ID, "state is true, but stopwatch does not have a start time")
				return nil
			}

			spo.StartTime = &msg.Stopwatch.StartTime
		} else {
			if !msg.Stopwatch.StartTime.IsZero() {
				spo.StartTime = &msg.Stopwatch.StartTime
			} else {
				log.Info("state is false and start time is empty")
			}

		}

		err := storage.Create(ctx, id, spo)
		if err != nil {
			return err
		}

		log.Info("creating new sewagepumpingstation in storage", "id", spo.ID)

		err = msgCtx.PublishOnTopic(ctx, spo)
		if err != nil {
			log.Error("failed to publish new sewagepumpingstation message")
			return err
		}

		log.Info("published message with id %s on topic %s", spo.ID, spo.TopicName())

	} else {
		spo, err := database.Get[SewagePumpingStation](ctx, storage, id)
		if err != nil {
			return err
		}
		if spo.State != msg.Stopwatch.State {
			if msg.Stopwatch.State {
				spo.State = msg.Stopwatch.State

				if msg.Stopwatch.StartTime.IsZero() {
					log.Error("invalid stopwatch message", msg.ID, "state is true, but stopwatch does not have a start time")
					return nil
				}

				spo.StartTime = &msg.Stopwatch.StartTime
				spo.ObservedAt = &msg.Timestamp

				storage.Update(ctx, id, spo)
				log.Info("updating sewagepumpingstation in storage", "id", spo.ID)

				err = msgCtx.PublishOnTopic(ctx, spo)
				if err != nil {
					return fmt.Errorf("failed to publish updated sewagepumpingstation message: %s", err)
				}
				log.Info("published message with id %s on topic %s", spo.ID, spo.TopicName())

			} else {
				spo.State = msg.Stopwatch.State

				spo.ObservedAt = &msg.Timestamp

				if spo.EndTime != nil && msg.Stopwatch.StopTime != nil {
					spo.EndTime = msg.Stopwatch.StopTime
				}

				storage.Update(ctx, id, spo)

				err = msgCtx.PublishOnTopic(ctx, spo)
				if err != nil {
					return fmt.Errorf("failed to publish updated sewagepumpingstation message: %s", err)
				}
				log.Info("published message with id %s on topic %s", spo.ID, spo.TopicName())
			}
		} else if spo.State == msg.Stopwatch.State {
			spo.ObservedAt = &msg.Timestamp
			storage.Update(ctx, id, spo)

			err = msgCtx.PublishOnTopic(ctx, spo)
			if err != nil {
				return fmt.Errorf("failed to publish updated sewagepumpingstation message: %s", err)
			}
			log.Info("published message with id %s on topic %s", spo.ID, spo.TopicName())
		}
	}

	return nil
}
