package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/umeshlumbhani/go-wifi-connect/internal/models"
	"github.com/umeshlumbhani/go-wifi-connect/internal/modules/command"
	"github.com/umeshlumbhani/go-wifi-connect/internal/modules/httpserver"
	"github.com/umeshlumbhani/go-wifi-connect/internal/modules/network"
)

func main() {
	cfg := models.NewConfig()
	logger := logrus.New()

	logger.SetFormatter(&logrus.TextFormatter{
		DisableSorting: true,
	})
	logger.SetLevel(logrus.InfoLevel)

	// ---------------------------Command -----------------------------
	cmd := command.NewCommand(logger, cfg)

	// --------------------------- Go Network Manager ------------------
	nw, err := network.NewNetwork(logger, cmd, cfg)
	if err != nil {
		panic(err)
	}

	// --------------------------- HTTP Server ------------------------
	httpServer := httpserver.NewHTTPServer(logger, nw, cfg)
	nw.HTTPServer = httpServer
	nw.StartPortal()

	// Setup stop signal handling
	signals := make(chan os.Signal, 1)
	exit := make(chan bool, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-signals
		logger.Info(fmt.Sprintf("Stop signal received, shutting down service (%v) ...", sig))
		nw.ClosePortal()
		exit <- true
	}()

	<-exit
	logger.Info("wifi-connect service stopped.")
}
