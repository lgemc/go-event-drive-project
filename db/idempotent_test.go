package db

import (
	"context"
	"github.com/ThreeDotsLabs/watermill"
	"os"
	"sync"
	"testing"
	"tickets/app"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"

	"tickets/app/repositories"
)

var db *sqlx.DB
var getDbOnce sync.Once

func TestIdempotent(t *testing.T) {
	db := getDb()

	err := app.Migrate(db)
	assert.NoError(t, err)
	repo := repositories.NewTicketsRepository(db)

	ticket := repositories.Ticket{
		TicketID:      watermill.NewUUID(),
		PriceCurrency: "USD",
	}

	err = repo.Put(context.Background(), ticket)
	assert.NoError(t, err)

	err = repo.Put(context.Background(), ticket)
	assert.NoError(t, err)

	tickets, err := repo.GetAll(context.Background())
	assert.NoError(t, err)

	assert.Len(t, tickets, 1)
}

func getDb() *sqlx.DB {
	getDbOnce.Do(func() {
		var err error
		db, err = sqlx.Open("postgres", os.Getenv("POSTGRES_URL"))
		if err != nil {
			panic(err)
		}
	})
	return db
}
