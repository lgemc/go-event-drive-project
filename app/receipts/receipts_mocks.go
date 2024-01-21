package receipts

import (
	"context"
	"sync"
	"time"
)

type IssueReceiptResponse struct {
	ReceiptNumber string    `json:"number"`
	IssuedAt      time.Time `json:"issued_at"`
}

type Service interface {
	IssueReceipt(ctx context.Context, request IssueReceiptRequest) (IssueReceiptResponse, error)
}

type ServiceMock struct {
	IssuedReceipts []IssueReceiptRequest

	receiptLock sync.Mutex
}

func (mock *ServiceMock) IssueReceipt(ctx context.Context, request IssueReceiptRequest) error {
	defer mock.receiptLock.Unlock()

	mock.receiptLock.Lock()

	mock.IssuedReceipts = append(mock.IssuedReceipts, request)

	return nil
}
