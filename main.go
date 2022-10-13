package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-redis/redis/v9"
)

var (
	// Version and date of the build. This is injected at build-time.
	buildString = "unknown"
	lo          = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)
	//go:embed assets/*
	assetsDir embed.FS
	//go:embed index.html
	html []byte
)

type App struct {
	redis *redis.Client
}

func main() {
	// Initialise and load the config.
	ko, err := initConfig()
	if err != nil {
		lo.Panic(err)
	}

	lo.Printf("booting cloak: %s\n", buildString)

	// Initialise connection.
	rdb := redis.NewClient(&redis.Options{
		Addr:     ko.String("redis.address"),
		Password: ko.String("redis.password"),
		DB:       ko.Int("redis.db"),
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		lo.Fatalf("unable to reach redis: %s", ko.String("redis.address"))
	}

	app := &App{
		redis: rdb,
	}

	// Register router instance.
	r := chi.NewRouter()

	// Register middlewares
	r.Use(middleware.Logger)

	// Frontend Handlers.
	assets, _ := fs.Sub(assetsDir, "assets")
	r.Get("/assets/*", func(w http.ResponseWriter, r *http.Request) {
		fs := http.StripPrefix("/assets/", http.FileServer(http.FS(assets)))
		fs.ServeHTTP(w, r)
	})
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")
		w.Write(html)
	})
	r.Get("/share/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")
		w.Write(html)
	})

	// API Handlers.
	r.Post("/api/encrypt", wrap(app, handleEncrypt))
	r.Get("/api/lookup/{uuid}", wrap(app, handleLookup))

	// HTTP Server.
	srv := &http.Server{
		Addr:         ko.String("server.address"),
		Handler:      r,
		ReadTimeout:  ko.Duration("server.timeout") * time.Millisecond,
		WriteTimeout: ko.Duration("server.timeout") * time.Millisecond,
		IdleTimeout:  ko.Duration("server.idle_timeout") * time.Millisecond,
	}

	lo.Printf("starting server: %s\n", srv.Addr)

	if err := srv.ListenAndServe(); err != nil {
		lo.Fatalf("couldn't start server: %v", err)
	}
}
