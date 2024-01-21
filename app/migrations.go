package app

import (
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const createTickets = `
CREATE TABLE IF NOT EXISTS tickets (
	ticket_id UUID PRIMARY KEY,
	price_amount NUMERIC(10, 2) NOT NULL,
	price_currency CHAR(3) NOT NULL,
	customer_email VARCHAR(255) NOT NULL
);
`

func migrate() error {
	db, err := sqlx.Open("postgres", os.Getenv("POSTGRES_URL"))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	_, err = db.Exec(createTickets)

	return err
}
