package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/vvolodya/go-metrics/internal/handler"
	"github.com/vvolodya/go-metrics/internal/repository"
)

func NewRouter(h *handler.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)

	r.Get("/", h.HandleIndex)
	r.Get("/value/{type}/{metric}", h.HandleMetric)
	r.Post("/update/{type}/{metric}/{value}", h.HandlePostMetrics)

	return r
}

func main() {
	s := repository.NewMemStorage()
	h := handler.NewHandler(s)
	r := NewRouter(h)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("http server is starting on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen and serve error: %v", err)
		}
	}()

	sig := <-stop
	log.Printf("signal received: %s — starting graceful shutdown", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown timeout: %v; forcing close", err)
		if cerr := srv.Close(); cerr != nil {
			log.Printf("server close error: %v", cerr)
		}
	}

	log.Println("server stopped cleanly")
}
