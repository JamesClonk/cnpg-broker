package main

import (
	"log"
	"net/http"
	"os"

	"github.com/cnpg-broker/pkg/broker"
	"github.com/gorilla/mux"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	b := broker.NewBroker()
	r := mux.NewRouter()

	r.HandleFunc("/v2/catalog", b.GetCatalog).Methods("GET")
	r.HandleFunc("/v2/service_instances/{instance_id}", b.Provision).Methods("PUT")
	r.HandleFunc("/v2/service_instances/{instance_id}", b.Deprovision).Methods("DELETE")
	r.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", b.Bind).Methods("PUT")
	r.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", b.Unbind).Methods("DELETE")

	log.Printf("CNPG Service Broker listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
