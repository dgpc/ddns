/*
    Copyright 2020 Google LLC

    Licensed under the Apache License, Version 2.0 (the "License");
    you may not use this file except in compliance with the License.
    You may obtain a copy of the License at

        https://www.apache.org/licenses/LICENSE-2.0

    Unless required by applicable law or agreed to in writing, software
    distributed under the License is distributed on an "AS IS" BASIS,
    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    See the License for the specific language governing permissions and
    limitations under the License.
*/

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"ddns/ddns"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/dns/v1"
)

func main() {
	var wg sync.WaitGroup
	defer wg.Wait() // application will not exit until WaitGroup empty

	c, err := google.DefaultClient(context.Background(), dns.CloudPlatformScope)
	if err != nil {
		log.Fatal(err)
	}
	dnsService, err := dns.New(c)
	if err != nil {
		log.Fatal(err)
	}

	ddnsServer := ddns.NewServer(dnsService)
	srv := &http.Server{Addr: ":8080", Handler: ddnsServer.Router}

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
		sig := <-sigs
		signal.Stop(sigs) // to allow force-exit on double ^C
		wg.Add(1)
		defer wg.Done()
		log.Printf("received %s, shutting down", sig)
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Printf("error whilst shutting down HTTP server: %v\n", err)
		}
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("error from ListenAndServe: %v\n", err)
	}
}
