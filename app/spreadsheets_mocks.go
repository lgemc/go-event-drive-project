package app

import (
	"context"
	"sync"
)

type SpreadsheetsClientMock struct {
	Sheets map[string][][]string // map[sheet_name][]row
	lock   sync.Mutex
}

func (c *SpreadsheetsClientMock) AppendRow(ctx context.Context, spreadsheetName string, row []string) error {
	c.Sheets[spreadsheetName] = append(c.Sheets[spreadsheetName], row)

	return nil
}
