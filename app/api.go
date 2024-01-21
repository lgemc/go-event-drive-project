package app

import (
	"context"
	"errors"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"net/http"
	"time"

	commonHTTP "github.com/ThreeDotsLabs/go-event-driven/common/http"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/labstack/echo/v4"
)

type TicketsRequest struct {
	Tickets []Ticket `json:"tickets"`
}

func handleTicket(ctx context.Context, ticket Ticket, bus *cqrs.EventBus) error {
	event := TicketEvent{
		Ticket: &ticket,
		Header: EventHeader{
			ID:          watermill.NewUUID(),
			PublishedAt: time.Now().Format(time.RFC3339Nano),
		},
	}

	switch ticket.Status {
	case TicketStatusConfirmed:
		return bus.Publish(ctx, TicketBookingConfirmed{
			TicketEvent: &TicketEvent{
				Ticket: event.Ticket,
				Header: event.Header,
			},
		})
	case TicketStatusCanceled:
		return bus.Publish(ctx, TicketCanceledEvent{
			TicketEvent: &TicketEvent{
				Ticket: event.Ticket,
				Header: event.Header,
			},
		})
	default:
		return errors.New("unknown ticket status")
	}
}

type NewServerInput struct {
	EventBus *cqrs.EventBus
	Logger   watermill.LoggerAdapter
}

func NewServer(input NewServerInput) *echo.Echo {
	e := commonHTTP.NewEcho()

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	e.POST("/tickets-status", func(c echo.Context) error {
		var request TicketsRequest
		err := c.Bind(&request)
		if err != nil {
			return err
		}

		correlationId := c.Request().Header.Get("Correlation-ID")
		if correlationId == "" {
			correlationId = watermill.NewUUID()
		}

		for _, ticket := range request.Tickets {
			err := handleTicket(context.Background(), ticket, input.EventBus)
			if err != nil {
				return c.String(http.StatusBadRequest, err.Error())
			}
		}

		return c.NoContent(http.StatusOK)
	})

	return e
}
