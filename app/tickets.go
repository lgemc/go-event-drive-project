package app

type TicketStatus string

func (t TicketStatus) String() string {
	return string(t)
}

const (
	TicketStatusConfirmed TicketStatus = "confirmed"
	TicketStatusCanceled  TicketStatus = "canceled"
)

type Price struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type Ticket struct {
	TicketID      string       `json:"ticket_id"`
	Status        TicketStatus `json:"status"`
	CustomerEmail string       `json:"customer_email"`
	Price         Price        `json:"price"`
}
