package app

type EventHeader struct {
	ID          string `json:"id"`
	PublishedAt string `json:"published_at"`
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
