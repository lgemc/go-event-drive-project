package app

import (
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/sirupsen/logrus"
)

type InjectHandlersInput struct {
	Router             *message.Router
	IssuesReceiptSub   *redisstream.Subscriber
	AppendToTrackerSub *redisstream.Subscriber
	ReceiptsClient     ReceiptsClientInterface
	SpreadsheetsClient SpreadsheetsClientInterface
}

func InjectHandlers(input InjectHandlersInput) {
	router := input.Router
	issueReceiptSub := input.IssuesReceiptSub
	receiptsClient := input.ReceiptsClient
	appendToTrackerSub := input.AppendToTrackerSub
	spreadsheetsClient := input.SpreadsheetsClient

	router.AddNoPublisherHandler(
		"issue_receipt",
		TicketBookingConfirmedTopic.String(),
		issueReceiptSub,
		func(msg *message.Message) error {
			ticket := Ticket{}
			err := json.Unmarshal(msg.Payload, &ticket)
			if err != nil {
				return err
			}

			return receiptsClient.IssueReceipt(msg.Context(), ticket)
		},
	)

	router.AddNoPublisherHandler(
		"print_ticket",
		TicketBookingConfirmedTopic.String(),
		appendToTrackerSub,
		func(msg *message.Message) error {
			ticket := Ticket{}
			err := json.Unmarshal(msg.Payload, &ticket)
			if err != nil {
				logrus.WithField("message_id", msg.UUID).Printf("error at unmarshal in print ticket: %v", err)
				return err
			}

			return spreadsheetsClient.AppendRow(msg.Context(), "tickets-to-print", []string{
				ticket.TicketID,
				ticket.CustomerEmail,
				ticket.Price.Amount,
				ticket.Price.Currency,
			})
		},
	)

	router.AddNoPublisherHandler(
		"append_canceled_ticket",
		TicketBookingCanceledTopic.String(),
		appendToTrackerSub,
		func(msg *message.Message) error {
			ticket := Ticket{}
			err := json.Unmarshal(msg.Payload, &ticket)
			if err != nil {
				return err
			}

			return spreadsheetsClient.AppendRow(msg.Context(), "tickets-to-refund", []string{
				ticket.TicketID,
				ticket.CustomerEmail,
				ticket.Price.Amount,
				ticket.Price.Currency,
			})
		},
	)
}
