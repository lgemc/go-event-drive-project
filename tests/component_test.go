package tests_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"tickets/app"
	"time"

	"github.com/lithammer/shortuuid/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//func TestServerIsUp(t *testing.T) {
//	t.Helper()
//	lock.Lock()
//	defer lock.Unlock()
//
//	a := waitForHttpServer(t)
//	a.Cancel()
//
//	<-a.Ctx.Done()
//}

//func TestIssuesReceipt(t *testing.T) {
//	t.Helper()
//
//	a := waitForHttpServer(t)
//	defer a.Cancel()
//	ticket := TicketStatus{
//		Status: app.TicketStatusConfirmed.String(),
//		Price: Money{
//			Amount:   "5",
//			Currency: "USD",
//		},
//	}
//	sendTicketsStatus(t, TicketsStatusRequest{
//		Tickets: []TicketStatus{ticket},
//	})
//
//	cli, ok := a.Dependencies.ReceiptsClient.(*app.ReceiptsServiceMock)
//	assert.True(t, ok)
//
//	assertReceiptForTicketIssued(t, cli, ticket)
//}

func TestErrorTicketsReceipt(t *testing.T) {
	t.Helper()

	a := waitForHttpServer(t)
	defer a.Cancel()

	ticket := TicketStatus{
		Status: app.TicketStatusCanceled.String(),
		Price: Money{
			Amount:   "5",
			Currency: "USD",
		},
	}
	sendTicketsStatus(t, TicketsStatusRequest{
		Tickets: []TicketStatus{ticket},
	})

	cli, ok := a.Dependencies.SpreadsheetsClient.(*app.SpreadsheetsClientMock)
	assert.True(t, ok)

	assertTrackTicketCanceled(t, cli, ticket)
}

func waitForHttpServer(t *testing.T) *app.App {
	t.Helper()
	_ = os.Setenv("GATEWAY_ADDR", "http://localhost:8000")

	a := app.NewApp(context.Background())

	err := a.InitMock()
	assert.NoError(t, err)

	go func() {
		a.Run()
	}()

	require.EventuallyWithT(
		t,
		func(t *assert.CollectT) {
			resp, err := http.Get("http://localhost:8080/health")
			if !assert.NoError(t, err) {
				return
			}
			defer resp.Body.Close()

			if assert.Less(t, resp.StatusCode, 300, "API not ready, http status: %d", resp.StatusCode) {
				return
			}
		},
		time.Second*10,
		time.Millisecond*50,
	)

	return a
}

type TicketsStatusRequest struct {
	Tickets []TicketStatus `json:"tickets"`
}

type TicketStatus struct {
	TicketID  string `json:"ticket_id"`
	Status    string `json:"status"`
	Price     Money  `json:"price"`
	Email     string `json:"email"`
	BookingID string `json:"booking_id"`
}

type Money struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

func assertTrackTicketCanceled(t *testing.T, spreadsheets *app.SpreadsheetsClientMock, ticket TicketStatus) {
	assert.EventuallyWithT(
		t,
		func(collectT *assert.CollectT) {
			errorSheet := spreadsheets.Sheets["tickets-to-refund"]
			errorRows := len(errorSheet)
			t.Log("issued receipts", errorRows)

			assert.Greater(collectT, errorRows, 0, "no receipts to refund")
		},
		10*time.Second,
		100*time.Millisecond,
	)

	column := spreadsheets.Sheets["tickets-to-refund"][0]
	assert.Len(t, column, 4)

	assert.Equal(t, ticket.TicketID, column[0])
	assert.Equal(t, ticket.Price.Amount, column[2])
	assert.Equal(t, ticket.Price.Currency, column[3])
}

func assertReceiptForTicketIssued(t *testing.T, receiptsService *app.ReceiptsServiceMock, ticket TicketStatus) {
	assert.EventuallyWithT(
		t,
		func(collectT *assert.CollectT) {
			issuedReceipts := len(receiptsService.IssuedReceipts)
			t.Log("issued receipts", issuedReceipts)

			assert.Greater(collectT, issuedReceipts, 0, "no receipts issued")
		},
		10*time.Second,
		100*time.Millisecond,
	)

	var receipt app.Ticket
	var ok bool
	for _, issuedReceipt := range receiptsService.IssuedReceipts {
		if issuedReceipt.TicketID != ticket.TicketID {
			continue
		}
		receipt = issuedReceipt
		ok = true
		break
	}
	require.Truef(t, ok, "receipt for ticket %s not found", ticket.TicketID)

	assert.Equal(t, ticket.TicketID, receipt.TicketID)
	assert.Equal(t, ticket.Price.Amount, receipt.Price.Amount)
	assert.Equal(t, ticket.Price.Currency, receipt.Price.Currency)
}

func sendTicketsStatus(t *testing.T, req TicketsStatusRequest) {
	t.Helper()

	payload, err := json.Marshal(req)
	require.NoError(t, err)

	correlationID := shortuuid.New()

	ticketIDs := make([]string, 0, len(req.Tickets))
	for _, ticket := range req.Tickets {
		ticketIDs = append(ticketIDs, ticket.TicketID)
	}

	httpReq, err := http.NewRequest(
		http.MethodPost,
		"http://localhost:8080/tickets-status",
		bytes.NewBuffer(payload),
	)
	require.NoError(t, err)

	httpReq.Header.Set("Correlation-ID", correlationID)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
