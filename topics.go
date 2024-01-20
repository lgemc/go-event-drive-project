package main

type TopicName string

func (t TopicName) String() string {
	return string(t)
}

const (
	TicketBookingConfirmedTopic TopicName = "TicketBookingConfirmed"
	TicketBookingCanceledTopic  TopicName = "TicketBookingCanceled"
)
