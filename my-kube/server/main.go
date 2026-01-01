package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"my-kube/pkg/server"
)

func main() {
	port := flag.Int("port", 8080, "Port to listen on")
	flag.Parse()

	fmt.Printf("Starting my-kube-server (Control Plane) on port %d...\n", *port)

	store := server.NewMemoryStore()
	apiServer := server.NewAPIServer(store)

	// Start Scheduler in background
	scheduler := server.NewScheduler(store)
	go scheduler.Run()

	// Start HTTP Server
	addr := fmt.Sprintf(":%d", *port)
	fmt.Printf("Listening on %s...\n", addr)
	log.Fatal(http.ListenAndServe(addr, apiServer)) // ServeHTTP method makes it a Handler
}
