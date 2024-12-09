package main

import (
	"context"
	"errors"
	database "forum/database"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	config, err := readServerConfig()
	if err != nil {
		return err
	}
	if err := database.InitDB(); err != nil {
		return err
	}
	defer database.DBInstance.DB.Close()
	server := newServer(config)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			log.Printf("Graceful shutdown: %v", err)
			server.Close()
		}
	}()
	log.Printf("Yaplane Community listening on https://%s", config.address)
	err = server.ListenAndServeTLS(config.certFile, config.keyFile)
	stop()
	<-stopped
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
