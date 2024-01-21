package app

import (
	"context"
	"net/http"
	"os"
	"os/signal"

	"golang.org/x/sync/errgroup"
)

type App struct {
	Dependencies *Dependencies
	ErrGroup     *errgroup.Group
	Ctx          context.Context
	Cancel       context.CancelFunc
}

func NewApp(ctx context.Context) *App {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)

	errgrp, ctx := errgroup.WithContext(ctx)

	return &App{
		ErrGroup: errgrp,
		Ctx:      ctx,
		Cancel:   cancel,
	}
}

func (a *App) InitMock() error {
	dependencies := &Dependencies{}

	err := dependencies.BuildMock()
	if err != nil {
		return err
	}

	a.Dependencies = dependencies

	return nil
}

func (a *App) Init() error {
	dependencies := &Dependencies{}

	err := dependencies.Build()
	if err != nil {
		return err
	}

	a.Dependencies = dependencies

	return nil
}

func (a *App) Run() {
	defer a.Cancel()

	ctx := a.Ctx

	errgrp := a.ErrGroup

	router := a.Dependencies.Router
	server := a.Dependencies.Server
	db := a.Dependencies.db

	errgrp.Go(func() error {
		// we don't want to start HTTP server before Watermill router (so service won't be healthy before it's ready)
		<-router.Running()

		err := server.Start(":8080")

		if err != nil && err != http.ErrServerClosed {
			return err
		}

		return nil
	})

	errgrp.Go(func() error {
		return router.Run(ctx)
	})

	// close
	errgrp.Go(func() error {
		<-ctx.Done()

		return router.Close()
	})

	errgrp.Go(func() error {
		<-ctx.Done()

		return db.Close()
	})

	errgrp.Go(func() error {
		<-ctx.Done()

		return server.Close()
	})

	err := errgrp.Wait()
	if err != nil && err != context.Canceled {
		panic(err)
	}
}
