package app

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/jmoiron/sqlx"
	"net/http"
	"os"
	"strconv"
	"tickets/app/repositories"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type Dependencies struct {
	ReceiptsClient     ReceiptsClientInterface
	SpreadsheetsClient SpreadsheetsClientInterface
	Router             *message.Router
	Server             *echo.Echo
	db                 *sqlx.DB
}

type BuildInput struct {
	ReceiptsClient     ReceiptsClientInterface
	SpreadsheetsClient SpreadsheetsClientInterface
}

func (d *Dependencies) Build() error {
	err := migrate()
	if err != nil {
		return err
	}

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
	spreadsheetsClient := NewSpreadsheetsClient(clients)

	return d.build(BuildInput{
		ReceiptsClient:     receiptsClient,
		SpreadsheetsClient: spreadsheetsClient,
	})
}

func (d *Dependencies) BuildMock() error {
	receiptsClient := ReceiptsServiceMock{}
	spreadsheetsClient := SpreadsheetsClientMock{
		Sheets: make(map[string][][]string),
	}

	return d.build(BuildInput{
		ReceiptsClient:     &receiptsClient,
		SpreadsheetsClient: &spreadsheetsClient,
	})
}

func (d *Dependencies) build(input BuildInput) error {
	db, err := sqlx.Open("postgres", os.Getenv("POSTGRES_URL"))
	if err != nil {
		return err
	}

	ticketsRepo := repositories.NewTicketsRepository(db)

	receiptsClient := input.ReceiptsClient
	spreadsheetsClient := input.SpreadsheetsClient

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

	bus, err := cqrs.NewEventBusWithConfig(pub, cqrs.EventBusConfig{
		Marshaler: cqrs.JSONMarshaler{GenerateName: cqrs.StructName},
		GeneratePublishTopic: func(params cqrs.GenerateEventPublishTopicParams) (string, error) {
			return params.EventName, nil
		},
		Logger: watermillLogger,
	})

	server := NewServer(NewServerInput{
		EventBus: bus,
		Logger:   watermillLogger,
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

	ep, err := cqrs.NewEventProcessorWithConfig(router, cqrs.EventProcessorConfig{
		SubscriberConstructor: func(params cqrs.EventProcessorSubscriberConstructorParams) (message.Subscriber, error) {
			return redisstream.NewSubscriber(redisstream.SubscriberConfig{
				Client:        rdb,
				ConsumerGroup: "svc-tickets." + params.HandlerName,
			}, watermillLogger)
		},
		Marshaler: cqrs.JSONMarshaler{
			GenerateName: cqrs.StructName,
		},
		GenerateSubscribeTopic: func(params cqrs.EventProcessorGenerateSubscribeTopicParams) (string, error) {
			return params.EventName, nil
		},
		Logger: watermillLogger,
	})
	if err != nil {
		return err
	}

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

	err = ep.AddHandlers(
		storeConfirmed,
		issuesReceipt,
		printTicket,
		appendCanceledTicket,
	)

	d.Router = router
	d.Server = server
	d.ReceiptsClient = receiptsClient
	d.SpreadsheetsClient = spreadsheetsClient

	return nil
}
