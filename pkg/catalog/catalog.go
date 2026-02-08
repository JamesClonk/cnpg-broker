package catalog

func GetCatalog() map[string]any {
	return map[string]any{
		"services": []map[string]any{
			{
				"id":          "79f7fb16-c95d-4210-8930-1c758648327e",
				"name":        "postgresql-dev-db",
				"description": "CloudNativePG PostgreSQL development database",
				"bindable":    true,
				"plans": []map[string]string{
					{
						"id":          "22cedd15-900f-4625-9f10-a57f43c64585",
						"name":        "dev-small",
						"description": "1 instance, 0.5 CPU, 512MB RAM, 10GB storage, no SLA",
					},
					{
						"id":          "de7acc66-412d-41c0-bf3e-763307a86c38",
						"name":        "dev-medium",
						"description": "1 instance, 2 CPU, 2GB RAM, 50GB storage, no SLA",
					},
					{
						"id":          "bfefc341-29a1-48e5-a6be-690f44aabbb3",
						"name":        "dev-large",
						"description": "1 instance, 4 CPU, 4GB RAM, 250GB storage, no SLA",
					},
				},
			},
			{
				"id":          "a651d10f-25ab-4a75-99a6-520c0abbe2ae",
				"name":        "postgresql-ha-cluster",
				"description": "CloudNativePG PostgreSQL database cluster",
				"bindable":    true,
				"plans": []map[string]string{
					{
						"id":          "9098f862-fb7e-42b5-9e8c-94c49e231cc3",
						"name":        "small",
						"description": "2 instances, 1 CPU, 1GB RAM, 10GB storage",
					},
					{
						"id":          "31aaeae1-4716-4631-b43e-93144e689427",
						"name":        "medium",
						"description": "3 instances, 2 CPU, 2GB RAM, 50GB storage",
					},
					{
						"id":          "b870dc08-1110-4bf8-ac82-e8a9d2bdd5c7",
						"name":        "large",
						"description": "3 instances, 4 CPU, 4GB RAM, 250GB storage",
					},
				},
			},
		},
	}
}

// returns (instances, cpu, memory, disk) based on planId
func PlanToSpec(planId string) (int64, string, string, string) {
	switch planId {
	case "22cedd15-900f-4625-9f10-a57f43c64585":
		return 1, "500m", "512Mi", "10Gi"
	case "de7acc66-412d-41c0-bf3e-763307a86c38":
		return 1, "2", "2Gi", "50Gi"
	case "bfefc341-29a1-48e5-a6be-690f44aabbb3":
		return 1, "4", "4Gi", "250Gi"
	case "9098f862-fb7e-42b5-9e8c-94c49e231cc3":
		return 3, "1", "1Gi", "10Gi"
	case "31aaeae1-4716-4631-b43e-93144e689427":
		return 3, "2", "2Gi", "50Gi"
	case "b870dc08-1110-4bf8-ac82-e8a9d2bdd5c7":
		return 3, "4", "4Gi", "250Gi"
	default:
		return 3, "4", "4Gi", "250Gi"
	}
}
