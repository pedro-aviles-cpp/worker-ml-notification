package main

import (
	"log"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"

	// This imports your root package so the init() function runs and registers the worker
	_ "github.com/pedro-aviles-cpp/worker-ml-notification"
)

func main() {
	// Defaulting to 8081 so it doesn't conflict with your fast-ingress container on 8080
	port := "8081"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	log.Printf("Starting Background Worker locally on port %s...", port)
	if err := funcframework.Start(port); err != nil {
		log.Fatalf("funcframework.Start: %v\n", err)
	}
}
