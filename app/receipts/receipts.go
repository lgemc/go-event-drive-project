package receipts

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/receipts"
)

type ReceiptsClientInterface interface {
	IssueReceipt(ctx context.Context, request IssueReceiptRequest) error
}

type ReceiptsClient struct {
	clients *clients.Clients
}

type Price struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type IssueReceiptRequest struct {
	TicketID       string `json:"ticket_id"`
	Status         string `json:"status"`
	CustomerEmail  string `json:"customer_email"`
	Price          Price  `json:"price"`
	IdempotencyKey string `json:"idempotency_kesy"`
}

func NewReceiptsClient(clients *clients.Clients) ReceiptsClientInterface {
	return ReceiptsClient{
		clients: clients,
	}
}

func (c ReceiptsClient) IssueReceipt(ctx context.Context, request IssueReceiptRequest) error {
	idempotencyKey := fmt.Sprintf("%s%s", request.IdempotencyKey, request.TicketID)
	body := receipts.PutReceiptsJSONRequestBody{
		IdempotencyKey: &idempotencyKey,
		TicketId:       request.TicketID,
		Price: receipts.Money{
			MoneyAmount:   request.Price.Amount,
			MoneyCurrency: request.Price.Currency,
		},
	}

	receiptsResp, err := c.clients.Receipts.PutReceiptsWithResponse(ctx, body)
	if err != nil {
		return err
	}
	if receiptsResp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status code: %v", receiptsResp.StatusCode())
	}

	return nil
}
