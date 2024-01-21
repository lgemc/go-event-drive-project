package app

import (
	"context"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/files"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/jmoiron/sqlx"
	"net/http"
	"os"
	"tickets/app/api"
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
	EventProcessor     *cqrs.EventProcessor
	Server             *echo.Echo
	db                 *sqlx.DB
}

type BuildInput struct {
	ReceiptsClient     ReceiptsClientInterface
	SpreadsheetsClient SpreadsheetsClientInterface
	FilesClient        files.ClientWithResponsesInterface
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
	spreadsheetsClient := NewSpreadsheetsClient(clients)

	err = d.build(BuildInput{
		ReceiptsClient:     receiptsClient,
		SpreadsheetsClient: spreadsheetsClient,
		FilesClient:        clients.Files,
	})
	if err != nil {
		return err
	}

	return Migrate(d.db)
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
	ticketsService := api.NewTicketsService(api.NewTicketsServiceInput{
		TicketRepository: ticketsRepo,
	})

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
		EventBus:       bus,
		Logger:         watermillLogger,
		TicketsService: ticketsService,
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

	err = injectHandlers(injectHandlersInput{
		receiptsClient:     receiptsClient,
		ticketsRepo:        ticketsRepo,
		spreadsheetsClient: spreadsheetsClient,
		filesClient:        input.FilesClient,
		eventBus:           bus,
	}, ep)
	if err != nil {
		return err
	}

	d.Router = router
	d.Server = server
	d.ReceiptsClient = receiptsClient
	d.SpreadsheetsClient = spreadsheetsClient
	d.db = db
	d.EventProcessor = ep

	return nil
}
