package app

import (
	"context"
	"net/http"
	"os"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type Dependencies struct {
	ReceiptsClient ReceiptsClientInterface
	Router         *message.Router
	Server         *echo.Echo
}

type BuildInput struct {
	ReceiptsClient ReceiptsClientInterface
	Clients        *clients.Clients
}

func (d *Dependencies) Build() error {
	clients, err := clients.NewClients(
		os.Getenv("GATEWAY_ADDR"),
		func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Correlation-ID", log.CorrelationIDFromContext(ctx))

			return nil
		})
	if err != nil {
		return err
	}

	receiptsClient := NewReceiptsClient(clients)

	return d.build(BuildInput{
		Clients:        clients,
		ReceiptsClient: receiptsClient,
	})
}

func (d *Dependencies) BuildMock() error {
	clients, err := clients.NewClients(
		os.Getenv("GATEWAY_ADDR"),
		func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Correlation-ID", log.CorrelationIDFromContext(ctx))

			return nil
		})
	if err != nil {
		return err
	}

	receiptsClient := ReceiptsServiceMock{}

	return d.build(BuildInput{
		Clients:        clients,
		ReceiptsClient: &receiptsClient,
	})
}

func (d *Dependencies) build(input BuildInput) error {
	clients := input.Clients
	receiptsClient := input.ReceiptsClient

	spreadsheetsClient := NewSpreadsheetsClient(clients)

	watermillLogger := log.NewWatermill(logrus.NewEntry(logrus.StandardLogger()))

	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})

	pub, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client: rdb,
	}, watermillLogger)
	if err != nil {
		return err
	}

	issueReceiptSub, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        rdb,
		ConsumerGroup: "issue-receipt",
	}, watermillLogger)
	if err != nil {
		return err
	}

	appendToTrackerSub, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        rdb,
		ConsumerGroup: "append-to-tracker",
	}, watermillLogger)
	if err != nil {
		return err
	}

	server := NewServer(NewServerInput{
		Pub: pub,
	})

	router, err := NewRouter(NewRouterInput{
		Logger: watermillLogger,
		Config: message.RouterConfig{},
	})
	if err != nil {
		return err
	}

	InjectMiddlewares(InjectMiddlewaresInput{
		Router: router,
		Logger: watermillLogger,
	})

	InjectHandlers(InjectHandlersInput{
		Router:             router,
		IssuesReceiptSub:   issueReceiptSub,
		AppendToTrackerSub: appendToTrackerSub,
		ReceiptsClient:     receiptsClient,
		SpreadsheetsClient: &spreadsheetsClient,
	})

	d.Router = router
	d.Server = server
	d.ReceiptsClient = receiptsClient

	return nil
}
