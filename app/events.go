package app

import (
	"github.com/google/uuid"
	"time"
)

type EventHeader struct {
	ID             string `json:"id"`
	PublishedAt    string `json:"published_at"`
	IdempotencyKey string `json:"idempotency_key"`
}

func NewEventHandlerWithIdempotencyKey(key string) EventHeader {
	return EventHeader{
		ID:             uuid.NewString(),
		PublishedAt:    time.Now().Format(time.RFC3339),
		IdempotencyKey: key,
	}
}

type TicketEvent struct {
	*Ticket
	Header EventHeader `json:"header"`
}

type TicketBookingConfirmed struct {
	*TicketEvent
}

type TicketCanceledEvent struct {
	*TicketEvent
}

type TicketPrinted struct {
	Header EventHeader `json:"header"`

	TicketID string `json:"ticket_id"`
	FileName string `json:"file_name"`
}
