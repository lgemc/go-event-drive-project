package repositories

import (
	"context"
	"github.com/jmoiron/sqlx"
)

/*
ticket_id UUID PRIMARY KEY,
price_amount NUMERIC(10, 2) NOT NULL,
price_currency CHAR(3) NOT NULL,
customer_email VARCHAR(255) NOT NULL
*/
type Ticket struct {
	TicketID      string  `db:"ticket_id"`
	PriceAmount   float64 `db:"price_amount"`
	PriceCurrency string  `db:"price_currency"`
	CustomerEmail string  `db:"customer_email"`
}

type TicketsRepository interface {
	Put(ctx context.Context, ticket Ticket) error
	Delete(ctx context.Context, ticketID string) error
	GetAll(ctx context.Context) ([]Ticket, error)
}

func NewTicketsRepository(db *sqlx.DB) TicketsRepository {
	return &ticketsRepository{
		db,
	}
}

type ticketsRepository struct {
	db *sqlx.DB
}

func (r *ticketsRepository) Put(ctx context.Context, ticket Ticket) error {
	_, err := r.db.NamedExec(`
INSERT INTO tickets 
    (ticket_id, price_amount, price_currency, customer_email)
VALUES  (:ticket_id, :price_amount, :price_currency, :customer_email)
ON CONFLICT DO NOTHING
`, ticket)

	return err
}

func (r *ticketsRepository) Delete(ctx context.Context, ticketID string) error {
	_, err := r.db.Exec("DELETE FROM tickets where ticket_id = $1", ticketID)

	return err
}

func (r *ticketsRepository) GetAll(ctx context.Context) ([]Ticket, error) {
	tickets := []Ticket{}

	err := r.db.SelectContext(ctx, &tickets, "SELECT * FROM tickets")
	if err != nil {
		return nil, err
	}

	return tickets, nil
}
