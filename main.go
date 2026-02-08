package main

import (
	"log"
	"os"

	"github.com/cnpg-broker/pkg/router"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// setup http router
	r := router.New()

	// serve API & UI
	log.Fatalf("failed to start HTTP router: %v", r.Start(8080)) // TODO: read from port/config above
}
