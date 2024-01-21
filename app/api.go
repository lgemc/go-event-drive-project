package app

import (
	"context"
	"errors"
	"fmt"
	commonHTTP "github.com/ThreeDotsLabs/go-event-driven/common/http"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"net/http"
	"tickets/app/api"
)

type TicketsRequest struct {
	Tickets []Ticket `json:"tickets"`
}

func handleTicket(ctx context.Context, ticket Ticket, idempotencyKey string, bus *cqrs.EventBus) error {
	event := TicketEvent{
		Ticket: &ticket,
		Header: NewEventHandlerWithIdempotencyKey(idempotencyKey),
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
	EventBus       *cqrs.EventBus
	TicketsService api.TicketsService
	Logger         watermill.LoggerAdapter
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

		idempotencyKey := c.Request().Header.Get("Idempotency-Key")
		if idempotencyKey == "" {
			idempotencyKey = fmt.Sprintf("gen_%s", uuid.NewString())
		}

		for _, ticket := range request.Tickets {
			err := handleTicket(context.Background(), ticket, idempotencyKey, input.EventBus)
			if err != nil {
				return c.String(http.StatusBadRequest, err.Error())
			}
		}

		return c.NoContent(http.StatusOK)
	})

	e.GET("/tickets", func(c echo.Context) error {
		tickets, err := input.TicketsService.GetAll(context.Background())
		if err != nil {
			return c.String(http.StatusInternalServerError, "internal error")
		}

		return c.JSON(http.StatusOK, tickets)
	})

	return e
}
