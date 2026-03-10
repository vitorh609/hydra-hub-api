package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"api-hydra-hub/internal/db"
	"api-hydra-hub/internal/httpx"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	router := httpx.NewRouter(pool)

	addr := ":" + port
	log.Printf("API rodando em %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server: %v", err)
	}
}
