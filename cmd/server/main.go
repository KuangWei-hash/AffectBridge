// Package main is the AffectBridge server entry point.
//
// AffectBridge is a Go service that connects a structured affective
// backend (initially ALMA) to a modern LLM to produce game characters
// with persistent, inspectable emotional state.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KuangWei-hash/AffectBridge/api"
	"github.com/KuangWei-hash/AffectBridge/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	mux := api.NewRouter(cfg)

	srv := &http.Server{
		Addr:        ":" + cfg.Port,
		Handler:     mux,
		ReadTimeout: 10 * time.Second,
		// Local reasoning models can spend well over 30 seconds on the
		// appraisal and expression calls in a single chat request.
		WriteTimeout: 180 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("affect-bridge listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
}
