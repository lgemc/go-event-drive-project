package tests_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"tickets/app"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComponent(t *testing.T) {
	// place for your tests!

	waitForHttpServer(t)
}

func waitForHttpServer(t *testing.T) {
	t.Helper()
	_ = os.Setenv("GATEWAY_ADDR", "http://localhost:8000")

	a := app.NewApp(context.Background())

	err := a.Init()
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
}
