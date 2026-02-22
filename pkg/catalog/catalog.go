package catalog

import (
	"os"

	"github.com/cnpg-broker/pkg/logger"
	"gopkg.in/yaml.v3"
)

var catalog Catalog

type Catalog struct {
	Services []Service `yaml:"services"`
}

type Service struct {
	ID                   string          `yaml:"id"`
	Name                 string          `yaml:"name"`
	Description          string          `yaml:"description"`
	Bindable             bool            `yaml:"bindable"`
	InstancesRetrievable bool            `yaml:"instances_retrievable"`
	BindingsRetrievable  bool            `yaml:"bindings_retrievable"`
	PlanUpdateable       bool            `yaml:"plan_updateable"`
	Tags                 []string        `yaml:"tags"`
	Metadata             ServiceMetadata `yaml:"metadata"`
	Plans                []Plan          `yaml:"plans"`
}

type ServiceMetadata struct {
	DisplayName         string `yaml:"displayName"`
	ImageUrl            string `yaml:"imageUrl"`
	LongDescription     string `yaml:"longDescription"`
	ProviderDisplayName string `yaml:"providerDisplayName"`
	DocumentationUrl    string `yaml:"documentationUrl"`
	SupportUrl          string `yaml:"supportUrl"`
}

type Plan struct {
	ID          string       `yaml:"id"`
	Name        string       `yaml:"name"`
	Description string       `yaml:"description"`
	Free        bool         `yaml:"free"`
	Metadata    PlanMetadata `yaml:"metadata"`
}

type PlanMetadata struct {
	Instances        int64  `yaml:"instances"`
	CPU              string `yaml:"cpu"`
	Memory           string `yaml:"memory"`
	Storage          string `yaml:"storage"`
	HighAvailability bool   `yaml:"highAvailability"`
	SLA              bool   `yaml:"sla"`
}

func init() {
	data, err := os.ReadFile("catalog.yaml")
	if err != nil {
		logger.Fatal("failed to read catalog.yaml: %v", err)
	}
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		logger.Fatal("failed to parse catalog.yaml: %v", err)
	}
	logger.Info("loaded catalog with %d service(s)", len(catalog.Services))
}

func GetCatalog() map[string]any {
	return map[string]any{"services": catalog.Services}
}

func PlanSpec(planId string) (int64, string, string, string) {
	for _, svc := range catalog.Services {
		for _, plan := range svc.Plans {
			if plan.ID == planId {
				logger.Debug("found plan %s: instances=%d, cpu=%s, memory=%s, storage=%s",
					planId, plan.Metadata.Instances, plan.Metadata.CPU,
					plan.Metadata.Memory, plan.Metadata.Storage)
				return plan.Metadata.Instances, plan.Metadata.CPU,
					plan.Metadata.Memory, plan.Metadata.Storage
			}
		}
	}
	logger.Warn("plan %s not found, using defaults", planId)
	return 3, "4", "4Gi", "250Gi"
}
