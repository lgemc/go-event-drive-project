package main

import (
	"context"
	"tickets/app"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/sirupsen/logrus"
)

func main() {
	log.Init(logrus.InfoLevel)

	a := app.NewApp(context.Background())

	err := a.Init()
	if err != nil {
		panic(err)
	}

	a.Run()
}
