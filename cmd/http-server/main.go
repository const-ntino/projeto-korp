package main

import (
	"log"
	"net/http"
	"time"

	"github.com/const-ntino/projeto-korp/internal/server"
)

func main() {
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           server.New(time.Now),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Println("http-server-projeto-korp listening on :8080")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
