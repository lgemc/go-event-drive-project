package app

import (
	"time"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

type NewRouterInput struct {
	Logger *log.WatermillLogrusAdapter
	Config message.RouterConfig
}

func NewRouter(input NewRouterInput) (*message.Router, error) {
	return message.NewRouter(input.Config, input.Logger)
}

type InjectMiddlewaresInput struct {
	Router *message.Router
	Logger *log.WatermillLogrusAdapter
}

func InjectMiddlewares(input InjectMiddlewaresInput) {
	router := input.Router

	// Middlewares
	router.AddMiddleware(injectCorrelationId)
	retry := middleware.Retry{
		MaxRetries:      10,
		InitialInterval: time.Millisecond * 100,
		MaxInterval:     time.Second,
		Multiplier:      2,
		Logger:          input.Logger,
	}

	logMiddleware := LogMiddleware{OkMessage: "Handling a message", ErrMessage: "Message handling error"}
	router.AddMiddleware(logMiddleware.Middleware)
	router.AddMiddleware(retry.Middleware)

	// skip messages without type because we don't want to handle them
	router.AddMiddleware(skipMessagesWithEmptyType)

	router.AddMiddleware(fixCurrency)
}
