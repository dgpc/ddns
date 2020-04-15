package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"ddns/ddns"
)

func main() {
	srv := &http.Server{Addr: ":8080", Handler: ddns.Router()}

	var wg sync.WaitGroup
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
		sig := <-sigs
		signal.Stop(sigs) // to allow force-exit on double ^C
		wg.Add(1)
		defer wg.Done()
		log.Printf("received %s, shutting down", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("error whilst shutting down HTTP server: %v\n", err)
		}
		cancel()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("error from ListenAndServe: %v\n", err)
		}
	}()

	wg.Wait() // wait for both ListenAndServe & Shutdown to return, as needed
}
