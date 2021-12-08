package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"mts.teta.mailsender/internal/app"
	"mts.teta.mailsender/internal/config"
)

const (
	appShutdownTimeout = 30 * time.Second
)

var (
	logger *zap.SugaredLogger
)

//@title Mailsender
//@version 0.1
//@description Service send mail to receivers.
//@termsOfService http://swagger.io/terms/
//@host petstore.swagger.io
//@BasePath /
func main() {
	log, _ := zap.NewProduction()
	logger = log.Sugar()

	cfg, err := config.FromEnv()
	if err != nil {
		logger.Fatal(err)
	}

	serverCtx, serverStopCtx := context.WithCancel(context.Background())
	application, err := app.New(context.Background(), cfg, logger)
	if err != nil {
		logger.Fatal(err)
	}
	application.Start(context.Background())
	gracefulShutdownTreatment(application, serverCtx, serverStopCtx)

	if err != nil {
		logger.Fatal(err)
	}

	<-serverCtx.Done()
}

func gracefulShutdownTreatment(application *app.App, serverCtx context.Context, serverStopCtx context.CancelFunc) {
	// copy from https://github.com/go-chi/chi/blob/master/_examples/graceful/main.go

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig

		// Shutdown signal with grace period of 30 seconds
		shutdownCtx, _ := context.WithTimeout(serverCtx, appShutdownTimeout)

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				logger.Fatal("graceful shutdown timed out.. forcing exit.")
			}
		}()

		application.Stop(shutdownCtx)
		serverStopCtx()
	}()
}
