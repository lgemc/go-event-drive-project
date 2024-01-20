package app

import (
	"context"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
)

type IssueReceiptResponse struct {
	ReceiptNumber string    `json:"number"`
	IssuedAt      time.Time `json:"issued_at"`
}

type ReceiptsService interface {
	IssueReceipt(ctx context.Context, request Ticket) (IssueReceiptResponse, error)
}

type ReceiptsServiceMock struct {
	IssuedReceipts []Ticket

	receiptLock sync.Mutex
}

func (mock *ReceiptsServiceMock) IssueReceipt(ctx context.Context, request Ticket) (IssueReceiptResponse, error) {
	defer mock.receiptLock.Unlock()

	mock.receiptLock.Lock()

	mock.IssuedReceipts = append(mock.IssuedReceipts, request)

	return IssueReceiptResponse{
		ReceiptNumber: watermill.NewUUID(),
		IssuedAt:      time.Now(),
	}, nil
}
