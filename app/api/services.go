package api

import (
	"context"
	"strconv"
	"tickets/app/repositories"
)

type PriceDTO struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type TicketDTO struct {
	TicketID      string   `json:"ticket_id"`
	CustomerEmail string   `json:"customer_email"`
	Price         PriceDTO `json:"price"`
}
type TicketsService interface {
	GetAll(ctx context.Context) ([]TicketDTO, error)
}

func NewTicketFromRepo(repoTicket repositories.Ticket) TicketDTO {
	return TicketDTO{
		TicketID:      repoTicket.TicketID,
		CustomerEmail: repoTicket.CustomerEmail,
		Price: PriceDTO{
			Amount:   strconv.FormatFloat(repoTicket.PriceAmount, 'f', 2, 64),
			Currency: repoTicket.PriceCurrency,
		},
	}
}

type NewTicketsServiceInput struct {
	TicketRepository repositories.TicketsRepository
}

type ticketService struct {
	ticketRepository repositories.TicketsRepository
}

func NewTicketsService(input NewTicketsServiceInput) TicketsService {
	return &ticketService{
		ticketRepository: input.TicketRepository,
	}
}

func (s *ticketService) GetAll(ctx context.Context) ([]TicketDTO, error) {
	tickets, err := s.ticketRepository.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	ticketsDTO := make([]TicketDTO, 0, len(tickets))
	for _, ticket := range tickets {
		ticketsDTO = append(ticketsDTO, NewTicketFromRepo(ticket))
	}

	return ticketsDTO, err
}
