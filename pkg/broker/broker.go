package broker

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/cnpg-broker/pkg/cnpg"
	"github.com/gorilla/mux"
)

type Broker struct {
	client *cnpg.Client
}

func NewBroker() *Broker {
	return &Broker{
		client: cnpg.NewClient(),
	}
}

func (b *Broker) GetCatalog(w http.ResponseWriter, r *http.Request) {
	catalog := map[string]interface{}{
		"services": []map[string]interface{}{
			{
				"id":          "79f7fb16-c95d-4210-8930-1c758648327e",
				"name":        "postgresql-dev-db",
				"description": "CloudNativePG PostgreSQL development database",
				"bindable":    true,
				"plans": []map[string]interface{}{
					{
						"id":          "22cedd15-900f-4625-9f10-a57f43c64585",
						"name":        "dev-small",
						"description": "1 instance, 10Gi storage, no SLA",
					},
					{
						"id":          "de7acc66-412d-41c0-bf3e-763307a86c38",
						"name":        "dev-medium",
						"description": "1 instance, 50Gi storage, no SLA",
					},
					{
						"id":          "bfefc341-29a1-48e5-a6be-690f44aabbb3",
						"name":        "dev-large",
						"description": "1 instance, 250Gi storage, no SLA",
					},
				},
			},
			{
				"id":          "a651d10f-25ab-4a75-99a6-520c0abbe2ae",
				"name":        "postgresql-ha-cluster",
				"description": "CloudNativePG PostgreSQL database cluster",
				"bindable":    true,
				"plans": []map[string]interface{}{
					{
						"id":          "9098f862-fb7e-42b5-9e8c-94c49e231cc3",
						"name":        "small",
						"description": "2 instances, 10Gi storage",
					},
					{
						"id":          "31aaeae1-4716-4631-b43e-93144e689427",
						"name":        "medium",
						"description": "3 instances, 50Gi storage",
					},
					{
						"id":          "b870dc08-1110-4bf8-ac82-e8a9d2bdd5c7",
						"name":        "large",
						"description": "3 instances, 250Gi storage",
					},
				},
			},
		},
	}
	json.NewEncoder(w).Encode(catalog)
}

func (b *Broker) Provision(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceID := vars["instance_id"]

	var req struct {
		ServiceID string                 `json:"service_id"`
		PlanID    string                 `json:"plan_id"`
		Context   map[string]interface{} `json:"context"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	namespace, err := b.client.CreateCluster(context.Background(), instanceID, req.PlanID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"dashboard_url": namespace,
	})
}

func (b *Broker) Deprovision(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		http.Error(w, "namespace required", http.StatusBadRequest)
		return
	}

	err := b.client.DeleteCluster(context.Background(), namespace)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{})
}

func (b *Broker) Bind(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceID := vars["instance_id"]

	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		http.Error(w, "namespace required", http.StatusBadRequest)
		return
	}

	creds, err := b.client.GetCredentials(context.Background(), instanceID, namespace)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"credentials": creds,
	})
}

func (b *Broker) Unbind(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{})
}
