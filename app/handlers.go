package app

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/files"
	"strconv"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"tickets/app/repositories"
)

type injectHandlersInput struct {
	receiptsClient     ReceiptsClientInterface
	ticketsRepo        repositories.TicketsRepository
	spreadsheetsClient SpreadsheetsClientInterface
	filesClient        files.ClientWithResponsesInterface
	eventBus           *cqrs.EventBus
}

func injectHandlers(input injectHandlersInput, ep *cqrs.EventProcessor) error {
	receiptsClient := input.receiptsClient
	ticketsRepo := input.ticketsRepo
	spreadsheetsClient := input.spreadsheetsClient

	issuesReceipt := cqrs.NewEventHandler[TicketBookingConfirmed]("issues-receipt", func(ctx context.Context, event *TicketBookingConfirmed) error {
		return receiptsClient.IssueReceipt(ctx, *event.Ticket)
	})

	storeConfirmed := cqrs.NewEventHandler[TicketBookingConfirmed]("store-confirmed", func(ctx context.Context, event *TicketBookingConfirmed) error {
		priceAmount, err := strconv.ParseFloat(event.Price.Amount, 64)
		if err != nil {
			return err
		}

		return ticketsRepo.Put(ctx, repositories.Ticket{
			TicketID:      event.TicketID,
			PriceAmount:   priceAmount,
			PriceCurrency: event.Price.Currency,
			CustomerEmail: event.CustomerEmail,
		})
	})

	deleteCanceled := cqrs.NewEventHandler[TicketCanceledEvent]("remove-canceled", func(ctx context.Context, event *TicketCanceledEvent) error {
		return ticketsRepo.Delete(ctx, event.TicketID)
	})

	printTicket := cqrs.NewEventHandler[TicketBookingConfirmed]("print-ticket", func(ctx context.Context, event *TicketBookingConfirmed) error {
		ticket := event.Ticket

		return spreadsheetsClient.AppendRow(ctx, "tickets-to-print", []string{
			ticket.TicketID,
			ticket.CustomerEmail,
			ticket.Price.Amount,
			ticket.Price.Currency,
		})
	})

	appendCanceledTicket := cqrs.NewEventHandler[TicketCanceledEvent]("append-canceled", func(ctx context.Context, event *TicketCanceledEvent) error {
		ticket := event.Ticket

		return spreadsheetsClient.AppendRow(ctx, "tickets-to-refund", []string{
			ticket.TicketID,
			ticket.CustomerEmail,
			ticket.Price.Amount,
			ticket.Price.Currency,
		})
	})

	createConfirmationFile := cqrs.NewEventHandler[TicketBookingConfirmed]("create-confirmation-file", func(ctx context.Context, event *TicketBookingConfirmed) error {
		fileName := fmt.Sprintf("%s-ticket.html", event.TicketID)
		_, err := input.filesClient.PutFilesFileIdContentWithTextBodyWithResponse(
			ctx,
			fileName,
			"hi")
		if err != nil {
			return err
		}

		return input.eventBus.Publish(ctx, TicketPrinted{
			Header:   event.Header,
			TicketID: event.TicketID,
			FileName: fileName,
		})
	})

	return ep.AddHandlers(
		storeConfirmed,
		issuesReceipt,
		printTicket,
		appendCanceledTicket,
		deleteCanceled,
		createConfirmationFile,
	)
}
