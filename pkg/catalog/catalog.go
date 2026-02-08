package catalog

import (
	"os"

	"gopkg.in/yaml.v3"
)

var catalog Catalog

type Catalog struct {
	Services []Service `yaml:"services"`
}

type Service struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Bindable    bool   `yaml:"bindable"`
	Plans       []Plan `yaml:"plans"`
}

type Plan struct {
	ID          string       `yaml:"id"`
	Name        string       `yaml:"name"`
	Description string       `yaml:"description"`
	Metadata    PlanMetadata `yaml:"metadata"`
}

type PlanMetadata struct {
	Instances int64  `yaml:"instances"`
	CPU       string `yaml:"cpu"`
	Memory    string `yaml:"memory"`
	Storage   string `yaml:"storage"`
}

func init() {
	data, err := os.ReadFile("catalog.yaml")
	if err != nil {
		panic(err) // TODO: change to logger Fatal
	}
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		panic(err) // TODO: change to logger Fatal
	}
}

func GetCatalog() map[string]any {
	return map[string]any{"services": catalog.Services}
}

func PlanSpec(planId string) (int64, string, string, string) {
	for _, svc := range catalog.Services {
		for _, plan := range svc.Plans {
			if plan.ID == planId {
				return plan.Metadata.Instances, plan.Metadata.CPU, plan.Metadata.Memory, plan.Metadata.Storage
			}
		}
	}
	// default
	return 3, "4", "4Gi", "250Gi"
}
