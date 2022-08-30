package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/app-sre/git-sync-pull/pkg/handler"
)

const (
	SERVER_PORT   = "SERVER_PORT"
	AWS_S3_BUCKET = "AWS_S3_BUCKET"
)

func main() {
	port, exists := os.LookupEnv(SERVER_PORT)
	if !exists {
		log.Fatalf("Missing environment variable: %s", SERVER_PORT)
	}
	bucket, exists := os.LookupEnv(AWS_S3_BUCKET)
	if !exists {
		log.Fatalf("Missing environment variable: %s", AWS_S3_BUCKET)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	// init "router"
	mux := http.NewServeMux()
	// init custom handler object
	handler, err := handler.NewHandler(ctx, bucket)
	if err != nil {
		log.Fatal(err)
	}
	mux.HandleFunc("/sync", handler.Sync)

	// start http server
	// hard-coded at this time to always use default network address
	log.Println(fmt.Sprintf("HTTP server listening at %s", port))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), mux))
}
