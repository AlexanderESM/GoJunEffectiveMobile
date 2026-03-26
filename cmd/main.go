package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"subscriptions/internal/handler"
	"subscriptions/internal/repository"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Error("DATABASE_URL is not set")
		os.Exit(1)
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer db.Close()
	log.Info("connected to database")

	repo := repository.New(db)
	h := handler.New(repo, log)

	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Info("starting server", "addr", addr)
	if err := http.ListenAndServe(addr, h.Routes()); err != nil {
		log.Error("server error", "err", err)
		os.Exit(1)
	}
}
