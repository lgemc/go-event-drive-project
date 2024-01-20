package app

import (
	"encoding/json"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/sirupsen/logrus"
)

type LogMiddleware struct {
	OkMessage  string
	ErrMessage string
}

func (m LogMiddleware) Middleware(next message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		logger := log.FromContext(msg.Context())

		logger.
			WithField("message_uuid", msg.UUID).
			Info(m.OkMessage)

		messages, err := next(msg)
		if err != nil {
			logger.
				WithField("message_uuid", msg.UUID).
				WithField("error", err).
				Error(m.ErrMessage)
		}

		return messages, err
	}
}

// fixCurrency is a middleware that fixes the currency of the ticket price.
// we get a bug report that sometimes the currency is empty, but we know that
// the default currency is USD, so we can fix it here.
func fixCurrency(next message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		ticket := Ticket{}

		err := json.Unmarshal(msg.Payload, &ticket)
		if err != nil {
			return nil, err
		}

		if ticket.Price.Currency == "" {
			ticket.Price.Currency = "USD"
		}

		payload, err := json.Marshal(ticket)
		if err != nil {
			return nil, err
		}

		msg.Payload = payload

		return next(msg)
	}
}

// skipMessagesWithEmptyType is a middleware that skips messages that don't have a type because
// we can't process them.
func skipMessagesWithEmptyType(next message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		messageType := msg.Metadata.Get("type")
		if messageType == "" {
			logrus.WithField("message_uuid", msg.UUID).Error("skipping message due to missing message type")
			return nil, nil
		}

		return next(msg)
	}
}

func injectCorrelationId(next message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		correlationId := msg.Metadata.Get("correlation_id")
		if correlationId == "" {
			correlationId = watermill.NewUUID()
		}
		ctx := log.ContextWithCorrelationID(msg.Context(), correlationId)
		ctx = log.ToContext(ctx, logrus.WithFields(logrus.Fields{"correlation_id": correlationId}))
		msg.SetContext(ctx)

		return next(msg)
	}
}
