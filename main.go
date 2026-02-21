package main

import (
	"log"
	"os"
	"strconv"

	"github.com/cnpg-broker/pkg/router"
)

func main() {
	p := os.Getenv("PORT")
	if len(p) == 0 {
		p = "8080"
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		log.Fatalf("could not parse PORT: %v", err)
	}

	// setup http router
	r := router.New()

	// serve API & UI
	log.Fatalf("failed to start HTTP router: %v", r.Start(port)) // TODO: read from port/config above
}
