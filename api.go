package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	commonHTTP "github.com/ThreeDotsLabs/go-event-driven/common/http"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/labstack/echo/v4"
)

type TicketsRequest struct {
	Tickets []Ticket `json:"tickets"`
}

func handleTicket(ticket Ticket, pub message.Publisher, correlationId string) error {
	event := TicketEvent{
		Ticket: &ticket,
		Header: EventHeader{
			ID:          watermill.NewUUID(),
			PublishedAt: time.Now().Format(time.RFC3339Nano),
		},
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata.Set("correlation_id", correlationId)

	switch ticket.Status {
	case TicketStatusConfirmed:
		msg.Metadata.Set("type", TicketBookingConfirmedTopic.String())
		err := pub.Publish(TicketBookingConfirmedTopic.String(), msg)
		if err != nil {
			return err
		}
	case TicketStatusCanceled:
		msg.Metadata.Set("type", TicketBookingCanceledTopic.String())
		err := pub.Publish(TicketBookingCanceledTopic.String(), msg)
		if err != nil {
			return err
		}
	default:
		return errors.New("unknown ticket status")
	}

	return nil
}

type NewServerInput struct {
	Pub *redisstream.Publisher
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
			err := handleTicket(ticket, input.Pub, correlationId)
			if err != nil {
				return c.String(http.StatusBadRequest, err.Error())
			}
		}

		return c.NoContent(http.StatusOK)
	})

	return e
}
