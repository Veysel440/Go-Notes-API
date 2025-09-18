package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Veysel440/go-notes-api/internal/config"
	"github.com/Veysel440/go-notes-api/internal/db"
	"github.com/Veysel440/go-notes-api/internal/server"
)

func main() {
	cfg := config.Load()
	pool, migrCloser, err := db.OpenAndMigrate(cfg)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer migrCloser()
	defer pool.Close()

	srv := server.New(cfg, pool)
	httpSrv := srv.HTTPServer()

	go func() {
		log.Printf("api listening on :%s", cfg.Port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
	log.Println("stopped")
}
