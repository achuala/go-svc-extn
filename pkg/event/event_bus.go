package event

import (
	"context"
)

type Event[T any] struct {
	Subject     string
	ID          string
	ContentType string
	Entity      string
	EntityID    string
	Payload     T
	Meta        map[string]string
}

func NewEvent[T any](subject string, id string, entity string, entityID string, payload T, meta map[string]string) *Event[T] {
	return &Event[T]{
		Subject:  subject,
		ID:       id,
		Entity:   entity,
		EntityID: entityID,
		Payload:  payload,
		Meta:     meta,
	}
}

// Publisher interface for publishing events
type EventPublisher interface {
	Publish(ctx context.Context, event *Event[any]) error
}

type EventBus struct {
	EventPublisher
}

type EventBusImpl struct {
	Publisher EventPublisher
}

func NewEventBus(publisher EventPublisher) *EventBusImpl {
	return &EventBusImpl{
		Publisher: publisher,
	}
}

func (b *EventBusImpl) Publish(ctx context.Context, event *Event[any]) error {
	return b.Publisher.Publish(ctx, event)
}
