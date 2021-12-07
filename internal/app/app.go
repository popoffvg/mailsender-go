package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/caarlos0/env"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"mts.teta.mailsender/internal/config"
	"mts.teta.mailsender/internal/mail"
	"mts.teta.mailsender/internal/mailsender"
	"mts.teta.mailsender/internal/sender"
	"mts.teta.mailsender/internal/store"
	"mts.teta.mailsender/pkg/mongodb"
)

const (
	dbStartTimeout = 5 * time.Second
)

type App struct {
	logger     *zap.SugaredLogger
	db         *mongo.Database
	mailSender *sender.Sender
	queue      store.MailingQueue
	server     *mailsender.Server
}

func New(ctx context.Context, cfg *config.Config, log *zap.SugaredLogger) (*App, error) {
	var cfgMail mail.MailConfig
	err := env.Parse(&cfgMail)
	if err != nil {
		return nil, err
	}
	log.Info(cfgMail)
	dbCtx, _ := context.WithTimeout(ctx, dbStartTimeout)
	db, err := mongodb.NewClient(
		dbCtx,
		cfg.Mongo.Addr,
		cfg.Mongo.Port,
		cfg.Mongo.User,
		cfg.Mongo.Pwd,
		cfg.Mongo.DB,
		cfg.Mongo.Auth,
	)

	if err != nil {
		return nil, err
	}

	queue, err := store.NewMongoQueue(db, log)
	if err != nil {
		return nil, err
	}

	mailSender := sender.New(
		queue,
		log,
		mail.New(&cfgMail, log),
	)

	return &App{
		logger:     log,
		db:         db,
		mailSender: mailSender,
		server:     mailsender.New(cfg, log, queue, mailSender),
		queue:      queue,
	}, nil
}

func (app *App) Start(ctx context.Context) {
	go func() {
		err := app.server.Start()
		if !errors.Is(err, http.ErrServerClosed) {
			app.mailSender.Stop()
			app.logger.Fatal("Error while starting public endpoint:\n", err)
			return
		}
	}()

	go app.mailSender.Serve()
}

func (app *App) Stop(ctx context.Context) {
	_ = app.server.Stop(ctx)
	app.mailSender.Stop()
	_ = app.db.Client().Disconnect(ctx)
}
