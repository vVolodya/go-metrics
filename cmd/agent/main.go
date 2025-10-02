package main

import (
	"context"
	"log"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vvolodya/go-metrics/internal/agent"
	"github.com/vvolodya/go-metrics/internal/repository"
)

const (
	baseUrl        = "http://localhost:8080"
	pollInterval   = 2 * time.Second
	reportInterval = 10 * time.Second
)

func main() {
	memStorage := repository.NewMemStorage()
	collector := agent.NewRuntimeCollector(memStorage, agent.StdRuntime{}, rand.New(rand.NewPCG(1, 2)))
	sender := agent.NewHTTPSender(memStorage, baseUrl)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				collector.PollOnce()
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(reportInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := sender.ReportOnce(ctx); err != nil {
					log.Printf("report error: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	<-ctx.Done()
	sender.Client.CloseIdleConnections()

	log.Println("agent stopped")
}
